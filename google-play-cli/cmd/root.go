package cmd

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/rudrankriyam/google-play-cli/internal/cli/googleplay"
	"github.com/rudrankriyam/google-play-cli/internal/cli/shared"
)

var versionRequested bool

// RootCommand returns the standalone Google Play CLI root command.
func RootCommand(version string) *ffcli.Command {
	versionRequested = false

	root := &ffcli.Command{
		Name:       "gplay",
		ShortUsage: "gplay <subcommand> [flags]",
		ShortHelp:  "Standalone Google Play CLI for Android Publisher automation.",
		FlagSet:    flag.NewFlagSet("gplay", flag.ExitOnError),
		UsageFunc:  shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			googleplay.GooglePlayAuthCommand(),
			googleplay.GooglePlayConfigCommand(),
			googleplay.GooglePlayAppsCommand(),
			googleplay.GooglePlayListingsCommand(),
			googleplay.GooglePlayEditsCommand(),
			googleplay.GooglePlayTracksCommand(),
			googleplay.GooglePlayBundlesCommand(),
			googleplay.GooglePlayAPKsCommand(),
			googleplay.GooglePlayReleaseCommand(),
			googleplay.GooglePlayMetadataCommand(),
			VersionCommand(version),
		},
	}

	root.FlagSet.BoolVar(&versionRequested, "version", false, "Print version and exit")
	shared.BindRootFlags(root.FlagSet)

	root.Exec = func(ctx context.Context, args []string) error {
		if versionRequested {
			fmt.Fprintln(os.Stdout, version)
			return nil
		}
		if len(args) > 0 {
			fmt.Fprintf(os.Stderr, "Unknown command: %s\n", strings.TrimSpace(args[0]))
		}
		return flag.ErrHelp
	}

	return root
}

// VersionCommand returns the version subcommand.
func VersionCommand(version string) *ffcli.Command {
	return &ffcli.Command{
		Name:       "version",
		ShortUsage: "gplay version",
		ShortHelp:  "Print version information and exit.",
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			fmt.Fprintln(os.Stdout, version)
			return nil
		},
	}
}
