package appcmd

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/urfave/cli/v3"
)

func lockNormalizeAction(ctx context.Context, cmd *cli.Command) error {
	content, err := normalizedLock(cmd.String("cwd"), cmd.String("type"))
	if err != nil {
		return err
	}

	output := cmd.String("output")
	if output == "" {
		fmt.Fprint(cmd.Writer, content)
		return nil
	}

	outputPath := filepath.Join(absPath(cmd.String("cwd")), output)
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(outputPath, []byte(content), 0o644)
}

func lockHashAction(ctx context.Context, cmd *cli.Command) error {
	content, err := normalizedLock(cmd.String("cwd"), cmd.String("type"))
	if err != nil {
		return err
	}
	sum := sha256.Sum256([]byte(content))
	fmt.Fprintln(cmd.Writer, hex.EncodeToString(sum[:]))
	return nil
}

func normalizedLock(cwdValue string, lockType string) (string, error) {
	if lockType != "npm" {
		return "", fmt.Errorf("Unsupported lock type: %s", lockType)
	}
	return normalizeNpmLock(absPath(cwdValue))
}

func normalizeNpmLock(cwd string) (string, error) {
	lockPath := filepath.Join(cwd, "package-lock.json")
	lock, err := readJSONObject(lockPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("package-lock.json not found in %s", cwd)
		}
		return "", err
	}

	delete(lock, "name")
	delete(lock, "version")
	if packages, ok := lock["packages"].(map[string]any); ok {
		if rootPackage, ok := packages[""].(map[string]any); ok {
			delete(rootPackage, "name")
			delete(rootPackage, "version")
			delete(rootPackage, "license")
			delete(rootPackage, "bin")
		}
	}

	content, err := json.MarshalIndent(lock, "", "  ")
	if err != nil {
		return "", err
	}
	return string(content) + "\n", nil
}
