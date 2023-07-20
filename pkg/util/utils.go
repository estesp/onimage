package util

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
)

func RunCommand(workdir string, command []string) (string, error) {
	cmd := exec.Command(command[0], command[1:]...)
	cmd.Env = append(cmd.Env, fmt.Sprintf("HOME=%s", workdir))
	cmd.Dir = workdir
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func DatetimeFromDir(dir string) string {
	parts := strings.Split(dir, "/")
	timestamp := parts[len(parts)-1]
	datestr := parts[len(parts)-2]

	return fmt.Sprintf("%s @ %s:%s", datestr, timestamp[:2], timestamp[2:4])
}

func GetDateString() string {
	return time.Now().Format("2006-01-02")
}
