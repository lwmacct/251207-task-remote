package appcmd

import (
	"bytes"
	"context"
	"path/filepath"
	"testing"
)

func TestBumpSetUpdatesNpmFilesWithoutNpmOutput(t *testing.T) {
	cwd := t.TempDir()
	utilWriteTestJSON(t, filepath.Join(cwd, "package.json"), map[string]any{
		"name":    "sample",
		"version": "0.1.0",
		"dependencies": map[string]any{
			"leftpad": "1.0.0",
		},
	})
	utilWriteTestJSON(t, filepath.Join(cwd, "package-lock.json"), map[string]any{
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
	err := cmd.Run(context.Background(), []string{"251207-task-remote", "bump", "set", "--cwd", cwd, "v1.2.3"})
	if err != nil {
		t.Fatal(err)
	}
	if stdout.String() != "" {
		t.Fatalf("expected no command output, got %q", stdout.String())
	}

	packageJSON := utilReadTestJSON(t, filepath.Join(cwd, "package.json"))
	packageLock := utilReadTestJSON(t, filepath.Join(cwd, "package-lock.json"))
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

func TestBumpSetRejectsUnsupportedLockfileVersionWithoutPartialWrite(t *testing.T) {
	cwd := t.TempDir()
	utilWriteTestJSON(t, filepath.Join(cwd, "package.json"), map[string]any{
		"name":    "sample",
		"version": "0.1.0",
	})
	utilWriteTestJSON(t, filepath.Join(cwd, "package-lock.json"), map[string]any{
		"name":            "sample",
		"version":         "0.1.0",
		"lockfileVersion": 1,
	})

	err := New().Run(context.Background(), []string{"251207-task-remote", "bump", "set", "--cwd", cwd, "1.2.3"})
	if err == nil {
		t.Fatal("expected error")
	}

	packageJSON := utilReadTestJSON(t, filepath.Join(cwd, "package.json"))
	packageLock := utilReadTestJSON(t, filepath.Join(cwd, "package-lock.json"))
	if packageJSON["version"] != "0.1.0" || packageLock["version"] != "0.1.0" {
		t.Fatalf("files were partially updated: package=%v lock=%v", packageJSON["version"], packageLock["version"])
	}
}

func TestBumpNextDevRule(t *testing.T) {
	next, err := nextVersion(t.TempDir(), "3", "v0.7.260101", "main", "260630")
	if err != nil {
		t.Fatal(err)
	}
	if next != "v0.1.260630" {
		t.Fatalf("next = %s", next)
	}
}
