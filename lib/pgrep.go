package lib

import "os/exec"

func FindCurrentProcessId(program string) string {
	processId := ""
	resp, err := exec.Command("pgrep", program).Output()
	if err != nil {
		HandleError(err, "[findCurrentProcessId] pgrep command failed")
		return ""
	}
	processId = FilterNewLines(string(resp))
	return processId
}
