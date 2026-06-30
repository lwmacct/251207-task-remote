package appcmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/urfave/cli/v3"
)

func bumpSetAction(ctx context.Context, cmd *cli.Command) error {
	return setVersion(cmd.String("cwd"), cmd.String("type"), cmd.Args().First())
}

func bumpNextAction(ctx context.Context, cmd *cli.Command) error {
	level := cmd.Args().First()
	if level == "" {
		level = "3"
	}

	next, err := nextVersion(cmd.String("cwd"), level, cmd.String("tag"), cmd.String("branch"), cmd.String("date"))
	if err != nil {
		return err
	}
	fmt.Fprintln(cmd.Writer, next)
	return nil
}

func setVersion(cwdValue string, versionType string, version string) error {
	if version == "" {
		return errors.New("bump set requires a version")
	}

	cwd := utilAbsPath(cwdValue)
	normalized := strings.TrimPrefix(version, "v")
	types, err := resolveVersionTypes(versionType, cwd)
	if err != nil {
		return err
	}

	for _, typ := range types {
		switch typ {
		case "npm":
			if err := setNpmVersion(cwd, normalized); err != nil {
				return err
			}
		case "python":
			if err := setPythonVersion(cwd, normalized); err != nil {
				return err
			}
		default:
			return fmt.Errorf("Unsupported version type: %s", typ)
		}
	}
	return nil
}

func resolveVersionTypes(versionType string, cwd string) ([]string, error) {
	if versionType == "npm" || versionType == "python" {
		return []string{versionType}, nil
	}
	if versionType != "all" && versionType != "auto" {
		return nil, fmt.Errorf("Unsupported version type: %s", versionType)
	}

	var types []string
	if utilFileExists(filepath.Join(cwd, "package.json")) {
		types = append(types, "npm")
	}
	if utilFileExists(filepath.Join(cwd, "pyproject.toml")) {
		types = append(types, "python")
	}
	return types, nil
}

func setNpmVersion(cwd string, version string) error {
	packageJSONPath := filepath.Join(cwd, "package.json")
	if !utilFileExists(packageJSONPath) {
		return fmt.Errorf("package.json not found in %s", cwd)
	}

	packageJSON, err := utilReadJSONObject(packageJSONPath)
	if err != nil {
		return err
	}
	packageJSON["version"] = version

	lockPath := filepath.Join(cwd, "package-lock.json")
	var packageLock map[string]any
	if utilFileExists(lockPath) {
		packageLock, err = utilReadJSONObject(lockPath)
		if err != nil {
			return err
		}
		lockfileVersion, ok := utilNumberAsInt(packageLock["lockfileVersion"])
		if !ok || !slices.Contains([]int{2, 3}, lockfileVersion) {
			return fmt.Errorf("Unsupported package-lock.json lockfileVersion: %v", packageLock["lockfileVersion"])
		}

		packageLock["version"] = version
		if packages, ok := packageLock["packages"].(map[string]any); ok {
			if rootPackage, ok := packages[""].(map[string]any); ok {
				rootPackage["version"] = version
			}
		}
	}

	if err := utilWriteJSONObject(packageJSONPath, packageJSON); err != nil {
		return err
	}
	if packageLock != nil {
		return utilWriteJSONObject(lockPath, packageLock)
	}
	return nil
}

func setPythonVersion(cwd string, version string) error {
	pyprojectPath := filepath.Join(cwd, "pyproject.toml")
	content, err := os.ReadFile(pyprojectPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("pyproject.toml not found in %s", cwd)
		}
		return err
	}

	re := regexp.MustCompile(`(?m)^version\s*=\s*["'][^"']*["']`)
	next := re.ReplaceAll(content, []byte(fmt.Sprintf(`version = "%s"`, version)))
	if bytes.Equal(next, content) {
		return fmt.Errorf("version field not found in %s", pyprojectPath)
	}
	return os.WriteFile(pyprojectPath, next, 0o644)
}

func nextVersion(cwdValue string, level string, tag string, branch string, date string) (string, error) {
	cwd := utilAbsPath(cwdValue)
	if tag == "" {
		tag = latestTag(cwd)
	}
	if branch == "" {
		branch = currentBranch(cwd)
	}

	major, minor, patch := parseVersion(tag)
	if strings.HasPrefix(branch, "dev/") || major == 0 {
		nextMinor := latestDevMinor(cwd) + 1
		if date == "" {
			date = utilShanghaiDate("060102")
		}
		return fmt.Sprintf("v0.%d.%s", nextMinor, date), nil
	}

	switch level {
	case "1", "major":
		return fmt.Sprintf("v%d.0.0", major+1), nil
	case "2", "minor":
		return fmt.Sprintf("v%d.%d.0", major, minor+1), nil
	default:
		return fmt.Sprintf("v%d.%d.%d", major, minor, patch+1), nil
	}
}

func parseVersion(tag string) (int, int, int) {
	parts := strings.Split(strings.TrimPrefix(tag, "v"), ".")
	return parsePart(parts, 0), parsePart(parts, 1), parsePart(parts, 2)
}

func parsePart(parts []string, index int) int {
	if index >= len(parts) {
		return 0
	}
	value, err := strconv.Atoi(parts[index])
	if err != nil {
		return 0
	}
	return value
}

func latestTag(cwd string) string {
	output := utilReadCommand(cwd, "git", "tag", "--sort=-v:refname")
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			return line
		}
	}
	return "v0.0.0"
}

func currentBranch(cwd string) string {
	branch := utilReadCommand(cwd, "git", "rev-parse", "--abbrev-ref", "HEAD")
	if branch == "" {
		return "main"
	}
	return branch
}

func latestDevMinor(cwd string) int {
	output := utilReadCommand(cwd, "git", "tag", "--sort=-v:refname")
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "v0.") {
			_, minor, _ := parseVersion(line)
			return minor
		}
	}
	return 0
}
