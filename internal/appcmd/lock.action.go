package appcmd

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/urfave/cli/v3"
)

func lockNormalizeAction(ctx context.Context, cmd *cli.Command) error {
	content, err := normalizedLock(cmd.String("cwd"), cmd.String("type"))
	if err != nil {
		return err
	}

	output := cmd.String("output")
	if output == "" {
		_, err := fmt.Fprint(cmd.Writer, content)
		return err
	}

	outputPath := filepath.Join(utilAbsPath(cmd.String("cwd")), output)
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
	_, err = fmt.Fprintln(cmd.Writer, hex.EncodeToString(sum[:]))
	return err
}

func normalizedLock(cwdValue string, lockType string) (string, error) {
	switch lockType {
	case "npm":
		return normalizeNpmLock(utilAbsPath(cwdValue))
	case "rust":
		return normalizeCargoLock(utilAbsPath(cwdValue))
	default:
		return "", fmt.Errorf("unsupported lock type: %s", lockType)
	}
}

func normalizeNpmLock(cwd string) (string, error) {
	lockPath := filepath.Join(cwd, "package-lock.json")
	lock, err := utilReadJSONObject(lockPath)
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

func normalizeCargoLock(cwd string) (string, error) {
	cargoTomlPath := filepath.Join(cwd, "Cargo.toml")
	cargoToml, err := os.ReadFile(cargoTomlPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("required file not found: Cargo.toml in %s", cwd)
		}
		return "", err
	}
	packageName, err := cargoPackageField(cargoToml, "name")
	if err != nil {
		return "", err
	}

	lockPath := filepath.Join(cwd, "Cargo.lock")
	lock, err := os.ReadFile(lockPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("required file not found: Cargo.lock in %s", cwd)
		}
		return "", err
	}
	content, err := normalizeCargoLockRootPackage(lock, packageName)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func cargoPackageField(content []byte, field string) (string, error) {
	fieldRE := regexp.MustCompile(fmt.Sprintf(`^\s*%s\s*=\s*["']([^"']+)["']`, regexp.QuoteMeta(field)))
	inPackage := false
	for _, line := range strings.Split(string(content), "\n") {
		section := tomlSectionName(line)
		if section != "" {
			inPackage = section == "package"
			continue
		}
		if !inPackage {
			continue
		}
		if matches := fieldRE.FindStringSubmatch(line); matches != nil {
			return matches[1], nil
		}
	}
	return "", fmt.Errorf("[package] %s field not found in Cargo.toml", field)
}

func setCargoPackageField(content []byte, field string, value string) ([]byte, error) {
	fieldRE := regexp.MustCompile(fmt.Sprintf(`^(\s*%s\s*=\s*)["'][^"']*["'](.*)$`, regexp.QuoteMeta(field)))
	lines := splitLinesAfter(string(content))
	inPackage := false
	updated := false
	for index, line := range lines {
		lineContent := strings.TrimSuffix(line, "\n")
		section := tomlSectionName(lineContent)
		if section != "" {
			inPackage = section == "package"
			continue
		}
		if !inPackage || !fieldRE.MatchString(lineContent) {
			continue
		}
		trailingNewline := ""
		if strings.HasSuffix(line, "\n") {
			trailingNewline = "\n"
		}
		lines[index] = fieldRE.ReplaceAllString(lineContent, `${1}"`+value+`"$2`) + trailingNewline
		updated = true
		break
	}
	if !updated {
		return nil, fmt.Errorf("[package] %s field not found in Cargo.toml", field)
	}
	return []byte(strings.Join(lines, "")), nil
}

func setCargoLockRootVersion(content []byte, packageName string, version string) ([]byte, error) {
	return rewriteCargoLockRootPackage(content, packageName, func(block []string) ([]string, error) {
		versionRE := regexp.MustCompile(`^(\s*version\s*=\s*)["'][^"']*["'](.*)$`)
		for index, line := range block {
			lineContent := strings.TrimSuffix(line, "\n")
			if !versionRE.MatchString(lineContent) {
				continue
			}
			trailingNewline := ""
			if strings.HasSuffix(line, "\n") {
				trailingNewline = "\n"
			}
			block[index] = versionRE.ReplaceAllString(lineContent, `${1}"`+version+`"$2`) + trailingNewline
			return block, nil
		}
		return nil, fmt.Errorf("version field for %s not found in Cargo.lock", packageName)
	})
}

func normalizeCargoLockRootPackage(content []byte, packageName string) ([]byte, error) {
	return rewriteCargoLockRootPackage(content, packageName, func(block []string) ([]string, error) {
		filtered := block[:0]
		for _, line := range block {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "name = ") || strings.HasPrefix(trimmed, "version = ") {
				continue
			}
			filtered = append(filtered, line)
		}
		return filtered, nil
	})
}

func rewriteCargoLockRootPackage(content []byte, packageName string, rewrite func([]string) ([]string, error)) ([]byte, error) {
	lines := splitLinesAfter(string(content))
	var output bytes.Buffer
	var block []string
	found := false

	flush := func() error {
		if len(block) == 0 {
			return nil
		}
		nextBlock := block
		if cargoLockPackageName(block) == packageName && !cargoLockPackageHasSource(block) {
			var err error
			nextBlock, err = rewrite(block)
			if err != nil {
				return err
			}
			found = true
		}
		for _, line := range nextBlock {
			output.WriteString(line)
		}
		block = nil
		return nil
	}

	for _, line := range lines {
		if strings.TrimSpace(line) == "[[package]]" {
			if err := flush(); err != nil {
				return nil, err
			}
		}
		block = append(block, line)
	}
	if err := flush(); err != nil {
		return nil, err
	}
	if !found {
		return nil, fmt.Errorf("root package %s not found in Cargo.lock", packageName)
	}
	return output.Bytes(), nil
}

func cargoLockPackageName(block []string) string {
	nameRE := regexp.MustCompile(`^\s*name\s*=\s*["']([^"']+)["']`)
	for _, line := range block {
		if matches := nameRE.FindStringSubmatch(line); matches != nil {
			return matches[1]
		}
	}
	return ""
}

func cargoLockPackageHasSource(block []string) bool {
	for _, line := range block {
		if strings.HasPrefix(strings.TrimSpace(line), "source = ") {
			return true
		}
	}
	return false
}

func tomlSectionName(line string) string {
	sectionRE := regexp.MustCompile(`^\s*\[{1,2}([A-Za-z0-9_.-]+)\]{1,2}\s*(?:#.*)?$`)
	if matches := sectionRE.FindStringSubmatch(line); matches != nil {
		return matches[1]
	}
	return ""
}

func splitLinesAfter(content string) []string {
	if content == "" {
		return nil
	}
	lines := strings.SplitAfter(content, "\n")
	if lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}
