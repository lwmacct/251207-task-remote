package appcmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/urfave/cli/v3"
	"gopkg.in/yaml.v3"
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
	_, err = fmt.Fprintln(cmd.Writer, next)
	return err
}

func setVersion(cwdValue string, versionType string, version string) error {
	if version == "" {
		return errors.New("bump set requires a version")
	}

	cwd := utilAbsPath(cwdValue)
	normalized := strings.TrimPrefix(version, "v")
	types, explicit, err := requestedVersionTypes(versionType)
	if err != nil {
		return err
	}

	var edits []versionFileEdit
	for _, typ := range types {
		var next []versionFileEdit
		switch typ {
		case "npm":
			next, err = collectNpmVersionEdits(cwd, normalized, explicit)
		case "python":
			next, err = collectPythonVersionEdits(cwd, normalized, explicit)
		case "rust":
			next, err = collectRustVersionEdits(cwd, normalized, explicit)
		default:
			return fmt.Errorf("unsupported version type: %s", typ)
		}
		if err != nil {
			return err
		}
		edits = append(edits, next...)
	}

	for _, edit := range dedupeVersionFileEdits(edits) {
		if err := os.WriteFile(edit.path, edit.content, 0o644); err != nil {
			return err
		}
	}
	return nil
}

type versionFileEdit struct {
	path    string
	content []byte
}

func requestedVersionTypes(versionType string) ([]string, bool, error) {
	if versionType == "npm" || versionType == "python" || versionType == "rust" {
		return []string{versionType}, true, nil
	}
	if versionType != "all" && versionType != "auto" {
		return nil, false, fmt.Errorf("unsupported version type: %s", versionType)
	}
	return []string{"npm", "python", "rust"}, false, nil
}

func dedupeVersionFileEdits(edits []versionFileEdit) []versionFileEdit {
	byPath := make(map[string]versionFileEdit, len(edits))
	for _, edit := range edits {
		byPath[edit.path] = edit
	}

	paths := make([]string, 0, len(byPath))
	for path := range byPath {
		paths = append(paths, path)
	}
	slices.Sort(paths)

	next := make([]versionFileEdit, 0, len(paths))
	for _, path := range paths {
		next = append(next, byPath[path])
	}
	return next
}

func collectNpmVersionEdits(cwd string, version string, require bool) ([]versionFileEdit, error) {
	packageJSONPaths, err := discoverNpmPackageJSONPaths(cwd)
	if err != nil {
		return nil, err
	}
	if len(packageJSONPaths) == 0 {
		if require {
			return nil, fmt.Errorf("package.json not found in %s", cwd)
		}
		return nil, nil
	}

	var edits []versionFileEdit
	var versionedPackageDirs []string
	for _, packageJSONPath := range packageJSONPaths {
		edit, hasVersion, err := collectPackageJSONVersionEdit(packageJSONPath, version)
		if err != nil {
			return nil, err
		}
		if !hasVersion {
			continue
		}
		edits = append(edits, edit)
		versionedPackageDirs = append(versionedPackageDirs, filepath.Dir(packageJSONPath))
	}
	if len(edits) == 0 {
		if require {
			return nil, fmt.Errorf("version field not found in npm packages under %s", cwd)
		}
		return nil, nil
	}

	lockEdit, ok, err := collectPackageLockVersionEdit(filepath.Join(cwd, "package-lock.json"), cwd, versionedPackageDirs, version)
	if err != nil {
		return nil, err
	}
	if ok {
		edits = append(edits, lockEdit)
	}
	return edits, nil
}

func collectPackageJSONVersionEdit(packageJSONPath string, version string) (versionFileEdit, bool, error) {
	packageJSONContent, err := os.ReadFile(packageJSONPath)
	if err != nil {
		return versionFileEdit{}, false, err
	}
	var packageJSON map[string]any
	if err := json.Unmarshal(packageJSONContent, &packageJSON); err != nil {
		return versionFileEdit{}, false, err
	}
	if _, ok := packageJSON["version"]; !ok {
		return versionFileEdit{}, false, nil
	}
	if _, ok := packageJSON["version"].(string); !ok {
		return versionFileEdit{}, false, fmt.Errorf("version field in %s is not a string", packageJSONPath)
	}
	nextPackageJSON, err := utilReplaceJSONStringValue(packageJSONContent, []string{"version"}, version)
	if err != nil {
		return versionFileEdit{}, false, err
	}
	return versionFileEdit{path: packageJSONPath, content: nextPackageJSON}, true, nil
}

func collectPackageLockVersionEdit(lockPath string, cwd string, packageDirs []string, version string) (versionFileEdit, bool, error) {
	if !utilFileExists(lockPath) {
		return versionFileEdit{}, false, nil
	}

	lockContent, err := os.ReadFile(lockPath)
	if err != nil {
		return versionFileEdit{}, false, err
	}
	var packageLock map[string]any
	if err := json.Unmarshal(lockContent, &packageLock); err != nil {
		return versionFileEdit{}, false, err
	}
	lockfileVersion, ok := utilNumberAsInt(packageLock["lockfileVersion"])
	if !ok || (lockfileVersion != 2 && lockfileVersion != 3) {
		return versionFileEdit{}, false, fmt.Errorf("unsupported package-lock.json lockfileVersion: %v", packageLock["lockfileVersion"])
	}

	if _, ok := packageLock["version"].(string); !ok {
		return versionFileEdit{}, false, fmt.Errorf("version field not found in %s", lockPath)
	}
	nextPackageLock, err := utilReplaceJSONStringValue(lockContent, []string{"version"}, version)
	if err != nil {
		return versionFileEdit{}, false, err
	}
	if packages, ok := packageLock["packages"].(map[string]any); ok {
		for _, packageDir := range packageDirs {
			rel, err := filepath.Rel(cwd, packageDir)
			if err != nil {
				return versionFileEdit{}, false, err
			}
			packageKey := filepath.ToSlash(rel)
			if packageKey == "." {
				packageKey = ""
			}
			lockedPackage, ok := packages[packageKey].(map[string]any)
			if !ok {
				continue
			}
			if _, ok := lockedPackage["version"].(string); ok {
				nextPackageLock, err = utilReplaceJSONStringValue(nextPackageLock, []string{"packages", packageKey, "version"}, version)
				if err != nil {
					return versionFileEdit{}, false, err
				}
			}
		}
	}
	return versionFileEdit{path: lockPath, content: nextPackageLock}, true, nil
}

func discoverNpmPackageJSONPaths(cwd string) ([]string, error) {
	packageDirs := map[string]bool{}
	if utilFileExists(filepath.Join(cwd, "package.json")) {
		packageDirs[cwd] = true
		patterns, err := readNpmWorkspacePackagePatterns(filepath.Join(cwd, "package.json"))
		if err != nil {
			return nil, err
		}
		if err := applyWorkspacePackagePatterns(cwd, packageDirs, patterns); err != nil {
			return nil, err
		}
	}

	workspacePath := filepath.Join(cwd, "pnpm-workspace.yaml")
	if utilFileExists(workspacePath) {
		patterns, err := readPnpmWorkspacePackagePatterns(workspacePath)
		if err != nil {
			return nil, err
		}
		if err := applyWorkspacePackagePatterns(cwd, packageDirs, patterns); err != nil {
			return nil, err
		}
	}

	paths := make([]string, 0, len(packageDirs))
	for packageDir := range packageDirs {
		paths = append(paths, filepath.Join(packageDir, "package.json"))
	}
	slices.Sort(paths)
	return paths, nil
}

func readNpmWorkspacePackagePatterns(packageJSONPath string) ([]string, error) {
	content, err := os.ReadFile(packageJSONPath)
	if err != nil {
		return nil, err
	}
	var packageJSON map[string]any
	if err := json.Unmarshal(content, &packageJSON); err != nil {
		return nil, err
	}
	return npmWorkspacePackagePatterns(packageJSON["workspaces"]), nil
}

func npmWorkspacePackagePatterns(value any) []string {
	switch typed := value.(type) {
	case []any:
		return stringSliceFromAny(typed)
	case map[string]any:
		if packages, ok := typed["packages"].([]any); ok {
			return stringSliceFromAny(packages)
		}
	}
	return nil
}

func stringSliceFromAny(values []any) []string {
	var stringsOnly []string
	for _, value := range values {
		if text, ok := value.(string); ok {
			stringsOnly = append(stringsOnly, text)
		}
	}
	return stringsOnly
}

func readPnpmWorkspacePackagePatterns(workspacePath string) ([]string, error) {
	content, err := os.ReadFile(workspacePath)
	if err != nil {
		return nil, err
	}
	var workspace struct {
		Packages []string `yaml:"packages"`
	}
	if err := yaml.Unmarshal(content, &workspace); err != nil {
		return nil, err
	}
	return workspace.Packages, nil
}

func applyWorkspacePackagePatterns(cwd string, packageDirs map[string]bool, patterns []string) error {
	for _, pattern := range patterns {
		exclude := strings.HasPrefix(pattern, "!")
		if exclude {
			pattern = strings.TrimSpace(strings.TrimPrefix(pattern, "!"))
		}
		matches, err := workspacePackageDirs(cwd, pattern)
		if err != nil {
			return err
		}
		for _, match := range matches {
			if exclude {
				delete(packageDirs, match)
			} else {
				packageDirs[match] = true
			}
		}
	}
	return nil
}

func workspacePackageDirs(cwd string, pattern string) ([]string, error) {
	pattern = filepath.ToSlash(strings.TrimSpace(pattern))
	if pattern == "" {
		return nil, nil
	}
	if filepath.IsAbs(pattern) || strings.HasPrefix(pattern, "../") || strings.Contains(pattern, "/../") || pattern == ".." {
		return nil, fmt.Errorf("unsupported pnpm workspace package pattern: %s", pattern)
	}

	var matches []string
	err := filepath.WalkDir(cwd, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}
		if path != cwd && (d.Name() == ".git" || d.Name() == "node_modules") {
			return filepath.SkipDir
		}
		if !utilFileExists(filepath.Join(path, "package.json")) {
			return nil
		}
		rel, err := filepath.Rel(cwd, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if rel == "." {
			rel = ""
		}
		ok, err := matchWorkspacePattern(pattern, rel)
		if err != nil {
			return err
		}
		if ok {
			matches = append(matches, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	slices.Sort(matches)
	return matches, nil
}

func matchWorkspacePattern(pattern string, rel string) (bool, error) {
	if pattern == "." {
		pattern = ""
	}
	patternParts := splitWorkspacePattern(pattern)
	relParts := splitWorkspacePattern(rel)
	return matchWorkspacePatternParts(patternParts, relParts)
}

func splitWorkspacePattern(value string) []string {
	if value == "" {
		return nil
	}
	return strings.Split(value, "/")
}

func matchWorkspacePatternParts(patternParts []string, relParts []string) (bool, error) {
	if len(patternParts) == 0 {
		return len(relParts) == 0, nil
	}
	if patternParts[0] == "**" {
		if len(patternParts) == 1 {
			return true, nil
		}
		for index := 0; index <= len(relParts); index++ {
			ok, err := matchWorkspacePatternParts(patternParts[1:], relParts[index:])
			if err != nil || ok {
				return ok, err
			}
		}
		return false, nil
	}
	if len(relParts) == 0 {
		return false, nil
	}
	ok, err := filepath.Match(patternParts[0], relParts[0])
	if err != nil || !ok {
		return ok, err
	}
	return matchWorkspacePatternParts(patternParts[1:], relParts[1:])
}

func collectPythonVersionEdits(cwd string, version string, require bool) ([]versionFileEdit, error) {
	pyprojectPath := filepath.Join(cwd, "pyproject.toml")
	content, err := os.ReadFile(pyprojectPath)
	if err != nil {
		if os.IsNotExist(err) {
			if require {
				return nil, fmt.Errorf("pyproject.toml not found in %s", cwd)
			}
			return nil, nil
		}
		return nil, err
	}

	re := regexp.MustCompile(`(?m)^version\s*=\s*["'][^"']*["']`)
	next := re.ReplaceAll(content, fmt.Appendf(nil, `version = "%s"`, version))
	if bytes.Equal(next, content) {
		return nil, fmt.Errorf("version field not found in %s", pyprojectPath)
	}
	return []versionFileEdit{{path: pyprojectPath, content: next}}, nil
}

func collectRustVersionEdits(cwd string, version string, require bool) ([]versionFileEdit, error) {
	cargoTomlPath := filepath.Join(cwd, "Cargo.toml")
	content, err := os.ReadFile(cargoTomlPath)
	if err != nil {
		if os.IsNotExist(err) {
			if require {
				return nil, fmt.Errorf("required file not found: Cargo.toml in %s", cwd)
			}
			return nil, nil
		}
		return nil, err
	}

	packageName, err := cargoPackageField(content, "name")
	if err != nil {
		return nil, err
	}
	next, err := setCargoPackageField(content, "version", version)
	if err != nil {
		return nil, err
	}

	lockPath := filepath.Join(cwd, "Cargo.lock")
	edits := []versionFileEdit{{path: cargoTomlPath, content: next}}
	if utilFileExists(lockPath) {
		lockContent, err := os.ReadFile(lockPath)
		if err != nil {
			return nil, err
		}
		nextLock, err := setCargoLockRootVersion(lockContent, packageName, version)
		if err != nil {
			return nil, err
		}
		edits = append(edits, versionFileEdit{path: lockPath, content: nextLock})
	}

	return edits, nil
}

func nextVersion(cwdValue string, level string, tag string, _ string, date string) (string, error) {
	cwd := utilAbsPath(cwdValue)
	if tag == "" {
		tag = latestTag(cwd)
	}

	major, minor, patch := parseVersion(tag)
	switch level {
	case "1", "major", "2", "minor", "3", "patch", "stable":
	default:
		return "", fmt.Errorf("unsupported version level: %s", level)
	}

	if major == 0 {
		if level == "stable" {
			return "v1.0.0", nil
		}
		nextMinor := latestZeroMajorMinor(cwd) + 1
		return fmt.Sprintf("v0.%d.%s", nextMinor, versionDate(date)), nil
	}

	switch level {
	case "1", "major":
		return fmt.Sprintf("v%d.0.0", major+1), nil
	case "2", "minor":
		return fmt.Sprintf("v%d.%d.0", major, minor+1), nil
	case "3", "patch":
		return fmt.Sprintf("v%d.%d.%d", major, minor, patch+1), nil
	case "stable":
		return "", fmt.Errorf("stable mode requires a v0 version, got %s", tag)
	default:
		return "", fmt.Errorf("unsupported version level: %s", level)
	}
}

func versionDate(value string) string {
	if value != "" {
		return value
	}
	return utilShanghaiDate("060102")
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

func latestZeroMajorMinor(cwd string) int {
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
