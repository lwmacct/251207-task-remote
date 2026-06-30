package appcmd

import (
	"encoding/json"
	"os"
	"testing"
)

func utilWriteTestJSON(t *testing.T, file string, value map[string]any) {
	t.Helper()
	content, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, append(content, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
}

func utilReadTestJSON(t *testing.T, file string) map[string]any {
	t.Helper()
	content, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	var value map[string]any
	if err := json.Unmarshal(content, &value); err != nil {
		t.Fatal(err)
	}
	return value
}
