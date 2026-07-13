package appcmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/urfave/cli/v3"
)

func gitDefaultBranchAction(ctx context.Context, cmd *cli.Command) error {
	_, err := fmt.Fprintln(cmd.Writer, defaultBranch())
	return err
}

func gitTopicBranchNameAction(ctx context.Context, cmd *cli.Command) error {
	_, err := fmt.Fprintln(cmd.Writer, topicBranchName(cmd.Args().Slice()))
	return err
}

func gitBackupBranchNameAction(ctx context.Context, cmd *cli.Command) error {
	name, err := backupBranchName(cmd.Args().First())
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(cmd.Writer, name)
	return err
}

func defaultBranch() string {
	originHead := utilReadCommand("", "git", "symbolic-ref", "refs/remotes/origin/HEAD")
	if strings.HasPrefix(originHead, "refs/remotes/origin/") {
		return strings.TrimPrefix(originHead, "refs/remotes/origin/")
	}

	remoteBranches := utilReadCommand("", "git", "branch", "-r")
	for _, branch := range []string{"main", "master"} {
		if utilContainsLine(remoteBranches, "origin/"+branch) {
			return branch
		}
	}
	return "main"
}

func topicBranchName(args []string) string {
	name := strings.Join(strings.Fields(strings.Join(args, " ")), "-")
	if name == "" {
		name = utilShanghaiDate("060102-1504")
	}
	return fmt.Sprintf("topic/%s", name)
}

func backupBranchName(branch string) (string, error) {
	if branch == "" {
		branch = utilReadCommand("", "git", "branch", "--show-current")
	}
	if branch == "" {
		return "", nil
	}

	date := utilShanghaiDate("0601021504")
	return fmt.Sprintf("backup/%s/%s/%s/%s/%s", date[0:2], date[2:4], date[4:6], date[6:], branch), nil
}
