package appcmd

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
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

func TestBumpSetPreservesNpmJSONLayout(t *testing.T) {
	cwd := t.TempDir()
	packageJSON := `{
    "scripts": {
        "test": "node test.js"
    },
    "version": "0.1.0",
    "name": "sample",
    "dependencies": {
        "leftpad": "1.0.0"
    }
}
`
	packageLock := `{
  "packages": {
    "node_modules/leftpad": {
      "version": "1.0.0"
    },
    "": {
      "dependencies": {
        "leftpad": "1.0.0"
      },
      "version": "0.1.0",
      "name": "sample"
    }
  },
  "requires": true,
  "lockfileVersion": 3,
  "version": "0.1.0",
  "name": "sample"
}
`
	if err := os.WriteFile(filepath.Join(cwd, "package.json"), []byte(packageJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cwd, "package-lock.json"), []byte(packageLock), 0o644); err != nil {
		t.Fatal(err)
	}

	err := New().Run(context.Background(), []string{"251207-task-remote", "bump", "set", "--cwd", cwd, "v1.2.3"})
	if err != nil {
		t.Fatal(err)
	}

	nextPackageJSON, err := os.ReadFile(filepath.Join(cwd, "package.json"))
	if err != nil {
		t.Fatal(err)
	}
	expectedPackageJSON := `{
    "scripts": {
        "test": "node test.js"
    },
    "version": "1.2.3",
    "name": "sample",
    "dependencies": {
        "leftpad": "1.0.0"
    }
}
`
	if string(nextPackageJSON) != expectedPackageJSON {
		t.Fatalf("package.json layout changed:\n%s", nextPackageJSON)
	}

	nextPackageLock, err := os.ReadFile(filepath.Join(cwd, "package-lock.json"))
	if err != nil {
		t.Fatal(err)
	}
	expectedPackageLock := `{
  "packages": {
    "node_modules/leftpad": {
      "version": "1.0.0"
    },
    "": {
      "dependencies": {
        "leftpad": "1.0.0"
      },
      "version": "1.2.3",
      "name": "sample"
    }
  },
  "requires": true,
  "lockfileVersion": 3,
  "version": "1.2.3",
  "name": "sample"
}
`
	if string(nextPackageLock) != expectedPackageLock {
		t.Fatalf("package-lock.json layout changed:\n%s", nextPackageLock)
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

func TestBumpSetUpdatesRustFiles(t *testing.T) {
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

	err := New().Run(context.Background(), []string{"251207-task-remote", "bump", "set", "--type", "rust", "--cwd", cwd, "v0.3.260630"})
	if err != nil {
		t.Fatal(err)
	}

	nextCargoToml, err := os.ReadFile(filepath.Join(cwd, "Cargo.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(nextCargoToml), `version = "0.3.260630"`) {
		t.Fatalf("Cargo.toml version was not updated: %s", nextCargoToml)
	}
	if !strings.Contains(string(nextCargoToml), `leftpad = "1.0.0"`) {
		t.Fatalf("dependency version was changed: %s", nextCargoToml)
	}

	nextCargoLock, err := os.ReadFile(filepath.Join(cwd, "Cargo.lock"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(nextCargoLock), "name = \"sample\"\nversion = \"0.3.260630\"") {
		t.Fatalf("Cargo.lock root version was not updated: %s", nextCargoLock)
	}
	if !strings.Contains(string(nextCargoLock), "name = \"leftpad\"\nversion = \"1.0.0\"") {
		t.Fatalf("Cargo.lock dependency version was changed: %s", nextCargoLock)
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
