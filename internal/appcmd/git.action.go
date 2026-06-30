package appcmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/urfave/cli/v3"
)

func gitDefaultBranchAction(ctx context.Context, cmd *cli.Command) error {
	fmt.Fprintln(cmd.Writer, defaultBranch())
	return nil
}

func gitDevBranchNameAction(ctx context.Context, cmd *cli.Command) error {
	fmt.Fprintln(cmd.Writer, devBranchName(cmd.Args().Slice()))
	return nil
}

func gitBackupBranchNameAction(ctx context.Context, cmd *cli.Command) error {
	name, err := backupBranchName(cmd.Args().First())
	if err != nil {
		return err
	}
	fmt.Fprintln(cmd.Writer, name)
	return nil
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
