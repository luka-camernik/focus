package lib

import (
	"fmt"
	"os/exec"
	"strings"
)

func Open(program string) {
	fmt.Println(fmt.Sprintf("[open] Opening %s", program))
	_ = exec.Command("/bin/sh", "-c", program, "%> /dev/null").Start()
}

func HandleError(err error, msg string) {
	if err != nil {
		fmt.Println(fmt.Sprintf("Error: %s ", msg), err)
	}
}

func FilterNewLines(s string) string {
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

func ReverseSlice(ss []string) []string {
	last := len(ss) - 1
	for i := 0; i < len(ss)/2; i++ {
		ss[i], ss[last-i] = ss[last-i], ss[i]
	}
	return ss
}

func CleanNames(s string) []string {
	var names []string
	split := strings.Split(string(s), ",")
	for _, str := range split {
		names = append(names, strings.Replace(str, "\"", "", -1))
	}
	return names
}

func SliceUniqueMap(s []string) []string {
	seen := make(map[string]struct{}, len(s))
	j := 0
	for _, v := range s {
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		s[j] = v
		j++
	}
	return s[:j]
}
