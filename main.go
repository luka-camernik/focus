package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"focus/lib"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"os/user"
	"strings"
	"sync"
	"time"
)

type Configuration struct {
	Program           string
	Open              bool
	ProcessId         string
	StackedWindows    []string
	WindowInformation []lib.Information
	AvailableWindows  []string
	CurrentWindowId   string
	ConfigFolder      string
	cacheFile         string
}

var configuration Configuration
var printHelp bool
var printVer bool
var start time.Time
var cacheTtl = time.Second * 300
var xprop lib.Xprop

func init() {
	start = time.Now()
	usr, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}
	configuration.ConfigFolder = usr.HomeDir + "/.config/focus/"
	os.Mkdir(configuration.ConfigFolder, 0700)
	configuration.cacheFile = configuration.ConfigFolder + "simplecache.json"
	var program string
	var open bool
	flag.Usage = printUsage
	flag.StringVar(&program, "p", "", "Which program to attempt to focus (Required)")
	flag.BoolVar(&open, "o", false, "Try to open the program if it cannot be found")
	flag.BoolVar(&printHelp, "help", false, "Print this help message.")
	flag.BoolVar(&printVer, "v", false, "Print version number.")
	flag.Parse()
	configuration.Program = program
	configuration.Open = open
}

func main() {
	if !checkDependencies() || !checkRequiredParams() {
		os.Exit(0)
	}
	preHeatCache()
	configuration.ProcessId = lib.FindCurrentProcessId(configuration.Program)
	fmt.Println(fmt.Sprintf("[Main] Current process ID is: %s", configuration.ProcessId))

	xprop.Root()
	configuration.StackedWindows = xprop.StackedWindows
	configuration.CurrentWindowId = xprop.CurrentWindowId
	fmt.Println(fmt.Sprintf("[Main] Current window ID is: %s", configuration.CurrentWindowId))

	if len(configuration.StackedWindows) > 0 {
		configuration.AvailableWindows = processWindowIds(configuration.StackedWindows, 0)
		cycle := false
		for _, wid := range configuration.AvailableWindows {
			if wid == configuration.CurrentWindowId {
				fmt.Println("[Main] This window is already in focus", configuration.CurrentWindowId)
				cycle = true
			}
		}
		if len(configuration.AvailableWindows) > 0 {
			attemptFocus(cycle)
		} else {
			if configuration.Open {
				fmt.Println("[Main] Trying to open a new window!")
				lib.Open(configuration.Program)
			} else {
				fmt.Println("[Main] No window found, and open config is not set")
			}
		}
	} else {
		fmt.Println("[Main] No stacked window found could be xprop error or missing xprop")
	}
	fmt.Println(fmt.Sprintf("[BENCHMARK] Command took %s to process", time.Since(start)))
}

func attemptFocus(cycle bool) {
	if cycle {
		// If we don't reverse them we only ever get max 2 windows 1 focused and 1 last opened
		// which resets because we just cycled
		configuration.AvailableWindows = lib.ReverseSlice(configuration.AvailableWindows)
		found := false
		for _, wid := range configuration.AvailableWindows {
			if wid != configuration.CurrentWindowId {
				found = true
				fmt.Println(fmt.Sprintf("[Main] Found another window of the program (%s)!", wid))
				if lib.Focus(wid) {
					break
				}
			}
		}
		if !found {
			fmt.Println("[Main] There is no other window to focus!")
		}
	} else {
		first := configuration.AvailableWindows[0]
		fmt.Println("[Main] Not cycling, focusing first available window", first)
		if !lib.Focus(first) {
			attemptFocus(true)
		}
	}
}

func processWindowIds(ids []string, round int) []string {
	if round > 2 {
		removeCache()
		fmt.Println("ERROR, there were too many rounds something went wrong!")
		return nil
	}
	var missing []string
	var availableWindows []string
	var wg sync.WaitGroup
	for _, windowId := range ids {
		wg.Add(1)
		go func(windowId string) {
			defer wg.Done()
			missingWindowId := true
			for _, info := range configuration.WindowInformation {
				t := time.Unix(info.Timestamp, 0)
				if time.Since(t) > cacheTtl {
					// Stale cache -> process it again
					continue
				}
				if info.WindowId == windowId {
					// It exists, not sure about being correct yet
					missingWindowId = false
					if configuration.ProcessId == info.ProcessId {
						availableWindows = append(availableWindows, windowId)
						break
					}
					for _, name := range info.Names {
						if strings.Contains(name, configuration.Program) {
							availableWindows = append(availableWindows, windowId)
							break
						}
					}
				}
			}
			if missingWindowId {
				missing = append(missing, windowId)
			}
		}(windowId)
	}
	wg.Wait()
	availableWindows = lib.SliceUniqueMap(availableWindows)

	if len(missing) > 0 {
		newWindows := xprop.Parse(configuration.StackedWindows)
		configuration.WindowInformation = xprop.WindowInformation
		if len(newWindows) > 0 {
			setCache(xprop.WindowInformation)
			return processWindowIds(newWindows, round+1)
		}
	}
	return availableWindows
}

func removeCache() {
	err := os.Remove(configuration.cacheFile)
	if err != nil {
		fmt.Println(err)
		return
	}
	configuration.WindowInformation = nil
}

func setCache(ids []lib.Information) {
	fmt.Println("Removing cache!")
	removeCache()
	fmt.Println("Setting cache!")
	config, _ := json.Marshal(ids)
	err := ioutil.WriteFile(configuration.cacheFile, config, 0644)
	if err != nil {
		fmt.Println(err)
		return
	}
}

func preHeatCache() {
	file, err := os.Open(configuration.cacheFile)
	if err != nil {
		return
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&configuration.WindowInformation)
	if err != nil {
		return
	}
}

func checkDependencies() bool {
	err := exec.Command("xprop", "-root").Run()
	if err != nil {
		fmt.Println("[DEPCHECK] Missing xprop dependency")
		return false
	}
	err = exec.Command("xdotool", "help").Run()
	if err != nil {
		fmt.Println("[DEPCHECK] Missing xdotool dependency")
		return false
	}
	return true
}

func checkRequiredParams() bool {
	if printHelp || printVer || configuration.Program == "" {
		printUsage()
		return false
	}
	return true
}

func printUsage() {
	fmt.Println("-----------------------------")
	fmt.Println("Focus v1.0.0")
	fmt.Println("-----------------------------")
	flag.PrintDefaults()
	fmt.Println("-----------------------------")
	fmt.Println("Dependencies xdotool and xprop")
	fmt.Println("-----------------------------")
}
