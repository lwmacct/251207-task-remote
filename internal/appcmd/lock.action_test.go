package appcmd

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
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

func TestLockNormalizeRust(t *testing.T) {
	cwd := t.TempDir()
	cargoToml := `[package]
name = "sample"
version = "0.1.0"
edition = "2024"

[dependencies]
leftpad = "1.0.0"
`
	cargoLock := `version = 4

[[package]]
name = "leftpad"
version = "1.0.0"
source = "registry+https://github.com/rust-lang/crates.io-index"
checksum = "abc"

[[package]]
name = "sample"
version = "0.1.0"
dependencies = [
 "leftpad",
]
`
	if err := os.WriteFile(filepath.Join(cwd, "Cargo.toml"), []byte(cargoToml), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cwd, "Cargo.lock"), []byte(cargoLock), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	cmd := New()
	cmd.Writer = &stdout
	err := cmd.Run(context.Background(), []string{"251207-task-remote", "lock", "normalize", "--type", "rust", "--cwd", cwd})
	if err != nil {
		t.Fatal(err)
	}

	content := stdout.String()
	if strings.Contains(content, `name = "sample"`) || strings.Contains(content, `version = "0.1.0"`) {
		t.Fatalf("normalized Cargo.lock still contains root metadata: %s", content)
	}
	if !strings.Contains(content, `name = "leftpad"`) || !strings.Contains(content, `version = "1.0.0"`) {
		t.Fatalf("normalized Cargo.lock removed dependency metadata: %s", content)
	}
}
