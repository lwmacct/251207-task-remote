package appcmd

import (
	"encoding/json"
	"os"
)

func readJSONObject(file string) (map[string]any, error) {
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

func writeJSONObject(file string, value map[string]any) error {
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

func numberAsInt(value any) (int, bool) {
	switch typed := value.(type) {
	case float64:
		return int(typed), typed == float64(int(typed))
	case int:
		return typed, true
	default:
		return 0, false
	}
}
