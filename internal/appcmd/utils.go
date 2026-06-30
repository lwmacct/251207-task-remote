package appcmd

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func utilReadJSONObject(file string) (map[string]any, error) {
	content, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	var value map[string]any
	if err := json.Unmarshal(content, &value); err != nil {
		return nil, err
	}
	return value, nil
}

func utilWriteJSONObject(file string, value map[string]any) error {
	original, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	content, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	if len(original) > 0 && original[len(original)-1] == '\n' {
		content = append(content, '\n')
	}
	return os.WriteFile(file, content, 0o644)
}

func utilNumberAsInt(value any) (int, bool) {
	switch typed := value.(type) {
	case float64:
		return int(typed), typed == float64(int(typed))
	case int:
		return typed, true
	default:
		return 0, false
	}
}

func utilReadCommand(cwd string, name string, args ...string) string {
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

func utilShanghaiDate(layout string) string {
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		location = time.FixedZone("CST", 8*60*60)
	}
	return time.Now().In(location).Format(layout)
}

func utilAbsPath(value string) string {
	if value == "" {
		value = "."
	}
	path, err := filepath.Abs(value)
	if err != nil {
		return value
	}
	return path
}

func utilFileExists(file string) bool {
	_, err := os.Stat(file)
	return err == nil
}

func utilContainsLine(output string, expected string) bool {
	for _, line := range strings.Split(output, "\n") {
		if strings.TrimSpace(line) == expected {
			return true
		}
	}
	return false
}
