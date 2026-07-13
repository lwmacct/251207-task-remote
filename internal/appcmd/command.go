package appcmd

import (
	"context"

	"github.com/lwmacct/251207-go-pkg-version/pkg/version"
	"github.com/urfave/cli/v3"
)

func New() *cli.Command {
	return &cli.Command{
		Name:            "251207-task-remote",
		Usage:           "task remote helper CLI",
		Version:         version.GetVersion(),
		HideHelpCommand: true,
		Commands: []*cli.Command{
			version.Command,
			bumpCommand(),
			lockCommand(),
			gitCommand(),
		},
		Action: showSubcommandHelpAction,
	}
}

func bumpCommand() *cli.Command {
	return &cli.Command{
		Name:  "bump",
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
				Action: bumpSetAction,
			},
			{
				Name:      "next",
				Usage:     "print next version",
				ArgsUsage: "[major|minor|patch|stable]",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "cwd", Value: "."},
					&cli.StringFlag{Name: "tag"},
					&cli.StringFlag{Name: "branch"},
					&cli.StringFlag{Name: "date"},
				},
				Action: bumpNextAction,
			},
		},
		Action: showSubcommandHelpAction,
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
				Name:   "normalize",
				Usage:  "print or write normalized lockfile content",
				Flags:  flags,
				Action: lockNormalizeAction,
			},
			{
				Name:   "hash",
				Usage:  "print normalized lockfile sha256",
				Flags:  flags,
				Action: lockHashAction,
			},
		},
		Action: showSubcommandHelpAction,
	}
}

func gitCommand() *cli.Command {
	return &cli.Command{
		Name:  "git",
		Usage: "print git helper values",
		Commands: []*cli.Command{
			{
				Name:   "default-branch",
				Usage:  "print default branch",
				Action: gitDefaultBranchAction,
			},
			{
				Name:      "topic-branch-name",
				Usage:     "print topic branch name",
				ArgsUsage: "[name...]",
				Action:    gitTopicBranchNameAction,
			},
			{
				Name:      "backup-branch-name",
				Usage:     "print backup branch name",
				ArgsUsage: "[branch]",
				Action:    gitBackupBranchNameAction,
			},
		},
		Action: showSubcommandHelpAction,
	}
}

func showSubcommandHelpAction(ctx context.Context, cmd *cli.Command) error {
	return cli.ShowSubcommandHelp(cmd)
}
