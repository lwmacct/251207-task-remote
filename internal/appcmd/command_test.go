package appcmd

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestVersionSetUpdatesNpmFilesWithoutNpmOutput(t *testing.T) {
	cwd := t.TempDir()
	writeTestJSON(t, filepath.Join(cwd, "package.json"), map[string]any{
		"name":    "sample",
		"version": "0.1.0",
		"dependencies": map[string]any{
			"leftpad": "1.0.0",
		},
	})
	writeTestJSON(t, filepath.Join(cwd, "package-lock.json"), map[string]any{
		"name":            "sample",
		"version":         "0.1.0",
		"lockfileVersion": 3,
		"requires":        true,
		"packages": map[string]any{
			"": map[string]any{
				"name":    "sample",
				"version": "0.1.0",
				"dependencies": map[string]any{
					"leftpad": "1.0.0",
				},
			},
			"node_modules/leftpad": map[string]any{
				"version": "1.0.0",
			},
		},
	})

	var stdout bytes.Buffer
	cmd := New()
	cmd.Writer = &stdout
	cmd.ErrWriter = &stdout
	err := cmd.Run(context.Background(), []string{"251207-task-remote", "version", "set", "--cwd", cwd, "v1.2.3"})
	if err != nil {
		t.Fatal(err)
	}
	if stdout.String() != "" {
		t.Fatalf("expected no command output, got %q", stdout.String())
	}

	packageJSON := readTestJSON(t, filepath.Join(cwd, "package.json"))
	packageLock := readTestJSON(t, filepath.Join(cwd, "package-lock.json"))
	if packageJSON["version"] != "1.2.3" {
		t.Fatalf("package.json version = %v", packageJSON["version"])
	}
	if packageLock["version"] != "1.2.3" {
		t.Fatalf("package-lock.json version = %v", packageLock["version"])
	}
	rootPackage := packageLock["packages"].(map[string]any)[""].(map[string]any)
	if rootPackage["version"] != "1.2.3" {
		t.Fatalf("package-lock root version = %v", rootPackage["version"])
	}
}

func TestVersionSetRejectsUnsupportedLockfileVersionWithoutPartialWrite(t *testing.T) {
	cwd := t.TempDir()
	writeTestJSON(t, filepath.Join(cwd, "package.json"), map[string]any{
		"name":    "sample",
		"version": "0.1.0",
	})
	writeTestJSON(t, filepath.Join(cwd, "package-lock.json"), map[string]any{
		"name":            "sample",
		"version":         "0.1.0",
		"lockfileVersion": 1,
	})

	err := New().Run(context.Background(), []string{"251207-task-remote", "version", "set", "--cwd", cwd, "1.2.3"})
	if err == nil {
		t.Fatal("expected error")
	}

	packageJSON := readTestJSON(t, filepath.Join(cwd, "package.json"))
	packageLock := readTestJSON(t, filepath.Join(cwd, "package-lock.json"))
	if packageJSON["version"] != "0.1.0" || packageLock["version"] != "0.1.0" {
		t.Fatalf("files were partially updated: package=%v lock=%v", packageJSON["version"], packageLock["version"])
	}
}

func TestLockNormalizeAndHash(t *testing.T) {
	cwd := t.TempDir()
	writeTestJSON(t, filepath.Join(cwd, "package-lock.json"), map[string]any{
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

func TestVersionNextDevRule(t *testing.T) {
	next, err := nextVersion(t.TempDir(), "3", "v0.7.260101", "main", "260630")
	if err != nil {
		t.Fatal(err)
	}
	if next != "v0.1.260630" {
		t.Fatalf("next = %s", next)
	}
}

func writeTestJSON(t *testing.T, file string, value map[string]any) {
	t.Helper()
	content, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, append(content, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readTestJSON(t *testing.T, file string) map[string]any {
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
