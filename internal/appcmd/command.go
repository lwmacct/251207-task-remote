package appcmd

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/urfave/cli/v3"
)

func New() *cli.Command {
	return &cli.Command{
		Name:            "251207-task-remote",
		Usage:           "task remote helper CLI",
		HideHelpCommand: true,
		Commands: []*cli.Command{
			versionCommand(),
			lockCommand(),
			gitCommand(),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return cli.ShowSubcommandHelp(cmd)
		},
	}
}

func versionCommand() *cli.Command {
	return &cli.Command{
		Name:  "version",
		Usage: "calculate and update project versions",
		Commands: []*cli.Command{
			{
				Name:      "set",
				Usage:     "update supported version files",
				ArgsUsage: "<version>",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "cwd", Value: "."},
					&cli.StringFlag{Name: "type", Value: "all"},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return setVersion(cmd.String("cwd"), cmd.String("type"), cmd.Args().First())
				},
			},
			{
				Name:      "next",
				Usage:     "print next version",
				ArgsUsage: "[level]",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "cwd", Value: "."},
					&cli.StringFlag{Name: "tag"},
					&cli.StringFlag{Name: "branch"},
					&cli.StringFlag{Name: "date"},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
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
				},
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return cli.ShowSubcommandHelp(cmd)
		},
	}
}

func lockCommand() *cli.Command {
	flags := []cli.Flag{
		&cli.StringFlag{Name: "cwd", Value: "."},
		&cli.StringFlag{Name: "type", Value: "npm"},
		&cli.StringFlag{Name: "output"},
	}

	return &cli.Command{
		Name:  "lock",
		Usage: "normalize lockfiles for cache keys",
		Commands: []*cli.Command{
			{
				Name:  "normalize",
				Usage: "print or write normalized lockfile content",
				Flags: flags,
				Action: func(ctx context.Context, cmd *cli.Command) error {
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
				},
			},
			{
				Name:  "hash",
				Usage: "print normalized lockfile sha256",
				Flags: flags,
				Action: func(ctx context.Context, cmd *cli.Command) error {
					content, err := normalizedLock(cmd.String("cwd"), cmd.String("type"))
					if err != nil {
						return err
					}
					sum := sha256.Sum256([]byte(content))
					fmt.Fprintln(cmd.Writer, hex.EncodeToString(sum[:]))
					return nil
				},
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return cli.ShowSubcommandHelp(cmd)
		},
	}
}

func gitCommand() *cli.Command {
	return &cli.Command{
		Name:  "git",
		Usage: "print git helper values",
		Commands: []*cli.Command{
			{
				Name:  "default-branch",
				Usage: "print default branch",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					fmt.Fprintln(cmd.Writer, defaultBranch())
					return nil
				},
			},
			{
				Name:      "dev-branch-name",
				Usage:     "print dev branch name",
				ArgsUsage: "[suffix...]",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					fmt.Fprintln(cmd.Writer, devBranchName(cmd.Args().Slice()))
					return nil
				},
			},
			{
				Name:      "backup-branch-name",
				Usage:     "print backup branch name",
				ArgsUsage: "[branch]",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					name, err := backupBranchName(cmd.Args().First())
					if err != nil {
						return err
					}
					fmt.Fprintln(cmd.Writer, name)
					return nil
				},
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return cli.ShowSubcommandHelp(cmd)
		},
	}
}

func setVersion(cwdValue string, versionType string, version string) error {
	if version == "" {
		return errors.New("version set requires a version")
	}

	cwd := absPath(cwdValue)
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
	if fileExists(filepath.Join(cwd, "package.json")) {
		types = append(types, "npm")
	}
	if fileExists(filepath.Join(cwd, "pyproject.toml")) {
		types = append(types, "python")
	}
	return types, nil
}

func setNpmVersion(cwd string, version string) error {
	packageJSONPath := filepath.Join(cwd, "package.json")
	if !fileExists(packageJSONPath) {
		return fmt.Errorf("package.json not found in %s", cwd)
	}

	packageJSON, err := readJSONObject(packageJSONPath)
	if err != nil {
		return err
	}
	packageJSON["version"] = version

	lockPath := filepath.Join(cwd, "package-lock.json")
	var packageLock map[string]any
	if fileExists(lockPath) {
		packageLock, err = readJSONObject(lockPath)
		if err != nil {
			return err
		}
		lockfileVersion, ok := numberAsInt(packageLock["lockfileVersion"])
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

	if err := writeJSONObject(packageJSONPath, packageJSON); err != nil {
		return err
	}
	if packageLock != nil {
		return writeJSONObject(lockPath, packageLock)
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
	cwd := absPath(cwdValue)
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
			date = shanghaiDate("060102")
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
	output := readCommand(cwd, "git", "tag", "--sort=-v:refname")
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			return line
		}
	}
	return "v0.0.0"
}

func currentBranch(cwd string) string {
	branch := readCommand(cwd, "git", "rev-parse", "--abbrev-ref", "HEAD")
	if branch == "" {
		return "main"
	}
	return branch
}

func latestDevMinor(cwd string) int {
	output := readCommand(cwd, "git", "tag", "--sort=-v:refname")
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "v0.") {
			_, minor, _ := parseVersion(line)
			return minor
		}
	}
	return 0
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

func defaultBranch() string {
	originHead := readCommand("", "git", "symbolic-ref", "refs/remotes/origin/HEAD")
	if strings.HasPrefix(originHead, "refs/remotes/origin/") {
		return strings.TrimPrefix(originHead, "refs/remotes/origin/")
	}

	remoteBranches := readCommand("", "git", "branch", "-r")
	for _, branch := range []string{"main", "master"} {
		if containsLine(remoteBranches, "origin/"+branch) {
			return branch
		}
	}
	return "main"
}

func devBranchName(args []string) string {
	suffix := strings.Join(args, " ")
	if suffix == "" {
		suffix = shanghaiDate("1504")
	}
	return fmt.Sprintf("dev/%s-%s", shanghaiDate("060102"), suffix)
}

func backupBranchName(branch string) (string, error) {
	if branch == "" {
		branch = readCommand("", "git", "branch", "--show-current")
	}
	if branch == "" {
		return "", nil
	}

	date := shanghaiDate("0601021504")
	return fmt.Sprintf("backup/%s/%s/%s/%s/%s", date[0:2], date[2:4], date[4:6], date[6:], branch), nil
}

func readJSONObject(file string) (map[string]any, error) {
	content, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	var value map[string]any
	if err := json.Unmarshal(content, &value); err != nil {
		return nil, err
	}
	return value, nil
}

func writeJSONObject(file string, value map[string]any) error {
	original, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	content, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	if len(original) > 0 && original[len(original)-1] == '\n' {
		content = append(content, '\n')
	}
	return os.WriteFile(file, content, 0o644)
}

func numberAsInt(value any) (int, bool) {
	switch typed := value.(type) {
	case float64:
		return int(typed), typed == float64(int(typed))
	case int:
		return typed, true
	default:
		return 0, false
	}
}

func readCommand(cwd string, name string, args ...string) string {
	cmd := exec.Command(name, args...)
	if cwd != "" {
		cmd.Dir = cwd
	}
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

func shanghaiDate(layout string) string {
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		location = time.FixedZone("CST", 8*60*60)
	}
	return time.Now().In(location).Format(layout)
}

func absPath(value string) string {
	if value == "" {
		value = "."
	}
	path, err := filepath.Abs(value)
	if err != nil {
		return value
	}
	return path
}

func fileExists(file string) bool {
	_, err := os.Stat(file)
	return err == nil
}

func containsLine(output string, expected string) bool {
	for _, line := range strings.Split(output, "\n") {
		if strings.TrimSpace(line) == expected {
			return true
		}
	}
	return false
}
