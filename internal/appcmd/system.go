package appcmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func readCommand(cwd string, name string, args ...string) string {
	cmd := exec.Command(name, args...)
	if cwd != "" {
		cmd.Dir = cwd
	}
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

func shanghaiDate(layout string) string {
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		location = time.FixedZone("CST", 8*60*60)
	}
	return time.Now().In(location).Format(layout)
}

func absPath(value string) string {
	if value == "" {
		value = "."
	}
	path, err := filepath.Abs(value)
	if err != nil {
		return value
	}
	return path
}

func fileExists(file string) bool {
	_, err := os.Stat(file)
	return err == nil
}

func containsLine(output string, expected string) bool {
	for _, line := range strings.Split(output, "\n") {
		if strings.TrimSpace(line) == expected {
			return true
		}
	}
	return false
}
