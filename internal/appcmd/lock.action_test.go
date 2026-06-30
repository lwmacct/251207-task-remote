package appcmd

import (
	"bytes"
	"context"
	"path/filepath"
	"testing"
)

func TestLockNormalizeAndHash(t *testing.T) {
	cwd := t.TempDir()
	utilWriteTestJSON(t, filepath.Join(cwd, "package-lock.json"), map[string]any{
		"name":            "sample",
		"version":         "0.1.0",
		"lockfileVersion": 3,
		"packages": map[string]any{
			"": map[string]any{
				"name":    "sample",
				"version": "0.1.0",
				"license": "ISC",
				"bin": map[string]any{
					"sample": "main.js",
				},
			},
		},
	})

	var stdout bytes.Buffer
	cmd := New()
	cmd.Writer = &stdout
	err := cmd.Run(context.Background(), []string{"251207-task-remote", "lock", "normalize", "--cwd", cwd})
	if err != nil {
		t.Fatal(err)
	}
	content := stdout.String()
	for _, forbidden := range []string{`"name"`, `"version"`, `"license"`, `"bin"`} {
		if bytes.Contains([]byte(content), []byte(forbidden)) {
			t.Fatalf("normalized lock still contains %s: %s", forbidden, content)
		}
	}
}
