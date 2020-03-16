package lib

import (
	"os/exec"
	"strings"
	"time"
)

type Xprop struct {
	StackedWindows    []string
	CurrentWindowId   string
	WindowInformation []Information
}

type Information struct {
	WindowId  string   `json:"window_id"`
	Names     []string `json:"names"`
	ProcessId string   `json:"process_id"`
	Timestamp int64    `json:"timestamp"`
}

func (xprop *Xprop) Root() {
	windowIds := ""
	resp, err := exec.Command("xprop", "-root").Output()
	split := strings.Split(string(resp), "\n")
	for _, str := range split {
		if strings.Contains(str, "_NET_CLIENT_LIST_STACKING(WINDOW)") {
			str = strings.Replace(str, "_NET_CLIENT_LIST_STACKING(WINDOW): window id # ", "", -1)
			windowIds = FilterNewLines(string(str))
			stackingIDs := strings.Split(windowIds, ", ")
			stackingIDs = ReverseSlice(stackingIDs) // We need to reverse because last focused was last in xprop -root
			xprop.StackedWindows = stackingIDs
		}
		if strings.Contains(str, "_NET_ACTIVE_WINDOW(WINDOW)") {
			str = strings.Replace(str, "_NET_ACTIVE_WINDOW(WINDOW): window id # ", "", -1)
			xprop.CurrentWindowId = FilterNewLines(string(str))
		}
	}
	if err != nil {
		HandleError(err, "[findStackingWindows] xprop -root command failed")
	}
}

func (xprop *Xprop) Parse(ids []string) []string {
	var parsedWindows []string
	for _, windowId := range ids {
		var info Information
		info.Timestamp = time.Now().Unix()
		info.WindowId = windowId
		resp, err := exec.Command("xprop", "-id", windowId).Output()
		if err != nil {
			HandleError(err, "[processWindowIds] xprop -id ... command failed")
			break
		}
		split := strings.Split(string(resp), "\n")
		for _, str := range split {
			if strings.Contains(str, "_NET_WM_PID(CARDINAL) = ") {
				str = strings.Replace(str, "_NET_WM_PID(CARDINAL) = ", "", -1)
				info.ProcessId = FilterNewLines(str)
			}
			if strings.Contains(str, "WM_CLASS(STRING) = ") {
				str = strings.Replace(str, "WM_CLASS(STRING) = ", "", -1)
				info.Names = CleanNames(FilterNewLines(str))
			}
		}
		xprop.WindowInformation = append(xprop.WindowInformation, info)
		parsedWindows = append(parsedWindows, windowId)
	}
	return parsedWindows
}
