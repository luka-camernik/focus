package main

import (
	"flag"
	"fmt"
	log "github.com/sirupsen/logrus"
	"os"
	"os/exec"
	"strings"
)

type Configuration struct {
	LogFile          string
	Program          string
	Open             bool
	ProgramId        string
	StackedWindows   []string
	AvailableWindows []string
	CurrentWindowId  string
}

var configuration Configuration
var printHelp bool
var printVer bool

func init() {
	var program string
	var logFile string
	var open bool
	flag.Usage = printUsage
	flag.StringVar(&program, "p", "", "Which program to attempt to focus (Required)")
	flag.BoolVar(&open, "o", false, "Try to open the program if it cannot be found")
	flag.StringVar(&logFile, "l", "", "Log file, defaults to console output if empty")
	flag.BoolVar(&printHelp, "help", false, "Print this help message.")
	flag.BoolVar(&printVer, "v", false, "Print version number.")
	flag.Parse()
	configuration.Program = program
	configuration.LogFile = logFile
	configuration.Open = open

	if configuration.LogFile == "" {
		log.SetOutput(os.Stdout)
	} else {
		f, err := os.OpenFile(configuration.LogFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			log.Fatalf("error opening file: %v", err)
		}
		log.SetOutput(f)
	}
	log.SetLevel(log.DebugLevel)

	customFormatter := new(log.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02 15:04:05"
	log.SetFormatter(customFormatter)
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
}

func main() {
	if !checkRequiredParams() {
		os.Exit(0)
	}
	configuration.ProgramId = findCurrentProgramId()
	log.Println(fmt.Sprintf("[Main] Current process ID is: %s", configuration.ProgramId))
	configuration.CurrentWindowId = findCurrentWindowId()
	log.Println(fmt.Sprintf("[Main] Current window ID is: %s", configuration.CurrentWindowId))
	configuration.StackedWindows = findStackingWindows()
	if len(configuration.StackedWindows) > 0 {
		configuration.AvailableWindows = processWindowIds(configuration.StackedWindows)
		cycle := false
		for _, wid := range configuration.AvailableWindows {
			if wid == configuration.CurrentWindowId {
				log.Println("[Main] This window is already in focus", configuration.CurrentWindowId)
				cycle = true
			}
		}
		if len(configuration.AvailableWindows) > 0 {
			if cycle {
				found := false
				for _, wid := range configuration.AvailableWindows {
					if wid != configuration.CurrentWindowId {
						found = true
						log.Println("[Main] Found another window of the program!", configuration.CurrentWindowId)
						focus(wid)
						continue
					}
				}
				if !found {
					log.Println("[Main] There is no other window to focus!")
				}
			} else {
				first := configuration.AvailableWindows[0]
				log.Println("[Main] Not cycling, focusing first available window", first)
				focus(first)
			}
		} else {
			if configuration.Open {
				log.Println("[Main] Trying to open a new window!")
				open(configuration.Program)
			} else {
				log.Println("[Main] No window found, and open config is not set")
			}
		}
	} else {
		log.Println("[Main] No stacked window found could be xprop error or missing xprop")
	}
}

func processWindowIds(ids []string) []string {
	var availableWindows []string
	for _, windowId := range ids {
		resp, err := exec.Command("/bin/bash", "-c", fmt.Sprintf("xprop -id %s | grep '_NET_WM_PID' | awk '{print $3}'", windowId)).Output()
		if err != nil {
			handleError(err, "[processWindowIds] xprop -id ... command failed")
			continue
		}
		programId := filterNewLines(string(resp))
		if programId == configuration.ProgramId {
			availableWindows = append(availableWindows, windowId)
		}
	}
	if len(availableWindows) == 0 {
		availableWindows = processWindowIdsByName(ids) // back fall to name parsing
	}
	return availableWindows
}

func processWindowIdsByName(ids []string) []string {
	var availableWindows []string
	for _, windowId := range ids {
		resp, err := exec.Command("/bin/bash", "-c", fmt.Sprintf("xprop -id %s | grep 'WM_CLASS(STRING)'", windowId)).Output()
		if err != nil {
			handleError(err, "[processWindowIds] xprop -id ... command failed")
			continue
		}
		wmClass := filterNewLines(string(resp))

		if strings.Contains(wmClass, configuration.Program) {
			availableWindows = append(availableWindows, windowId)
		}
	}
	return availableWindows
}

func findCurrentProgramId() string {
	programId := ""
	resp, err := exec.Command("pgrep", configuration.Program).Output()
	if err != nil {
		handleError(err, "[findCurrentProgramId] pgrep command failed")
		return ""
	}
	programId = filterNewLines(string(resp))
	return programId
}

func findCurrentWindowId() string {
	windowId := ""
	resp, err := exec.Command("/bin/bash", "-c", "xprop -root | grep '_NET_ACTIVE_WINDOW(WINDOW)' | awk -F'#' '{print $2}'").Output()
	if err != nil {
		handleError(err, "[findCurrentWindowId] xprop -root command failed")
		return ""
	}
	windowId = filterNewLines(string(resp))
	return windowId
}

func findStackingWindows() []string {
	windowIds := ""
	resp, err := exec.Command("/bin/bash", "-c", "xprop -root | grep '_NET_CLIENT_LIST_STACKING(WINDOW)' | awk -F'#' '{print $2}'").Output()
	if err != nil {
		handleError(err, "[findStackingWindows] xprop -root command failed")
		return nil
	}
	windowIds = filterNewLines(string(resp))
	stackingIDs := strings.Split(windowIds, ", ")
	stackingIDs = reverse(stackingIDs) // We need to reverse because last focused was last in xprop -root
	return stackingIDs
}

func open(program string) {
	fmt.Println(fmt.Sprintf("[open] Opening %s", program))
	exec.Command("/bin/sh", "-c", program, "%> /dev/null").Start()
}

func focus(wid string) {
	err := exec.Command("xdotool", "windowactivate", wid).Run()
	if err != nil {
		handleError(err, "[focus] xdotool command failed")
		return
	}
}

func handleError(err error, msg string) {
	if err != nil {
		log.Println(fmt.Sprintf("Error: %s ", msg), err)
	}
}

func filterNewLines(s string) string {
	s = strings.Map(func(r rune) rune {
		switch r {
		case 0x000A, 0x000B, 0x000C, 0x000D, 0x0085, 0x2028, 0x2029:
			return -1
		default:
			return r
		}
	}, s)
	s = strings.TrimSpace(s)
	return s
}

func reverse(ss []string) []string {
	last := len(ss) - 1
	for i := 0; i < len(ss)/2; i++ {
		ss[i], ss[last-i] = ss[last-i], ss[i]
	}
	return ss
}
