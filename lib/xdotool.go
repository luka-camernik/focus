package lib

import (
	"fmt"
	"os/exec"
)

func Focus(windowId string) bool {
	fmt.Println(fmt.Sprintf("[Focus] Focusing %s", windowId))
	err := exec.Command("xdotool", "windowactivate", windowId).Start()
	if err != nil {
		HandleError(err, "[focus] xdotool command failed")
		return false
	}
	return true
}
