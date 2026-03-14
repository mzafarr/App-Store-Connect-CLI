package googleplay

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/rudrankriyam/google-play-cli/internal/cli/shared"
)

// GooglePlayEditsCommand returns the edits command group.
func GooglePlayEditsCommand() *ffcli.Command {
	fs := flag.NewFlagSet("edits", flag.ExitOnError)

	return &ffcli.Command{
		Name:       "edits",
		ShortUsage: "gplay edits <subcommand> [flags]",
		ShortHelp:  "Create and commit Google Play edits.",
		LongHelp: `Create and commit Google Play edits.

Examples:
  gplay edits create --package-name "com.example.app"
  gplay edits commit --package-name "com.example.app" --edit-id "EDIT_ID"`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			GooglePlayEditsCreateCommand(),
			GooglePlayEditsCommitCommand(),
		},
		Exec: func(ctx context.Context, args []string) error {
			return flag.ErrHelp
		},
	}
}

// GooglePlayEditsCreateCommand returns the edits create subcommand.
func GooglePlayEditsCreateCommand() *ffcli.Command {
	fs := flag.NewFlagSet("edits create", flag.ExitOnError)

	packageName := fs.String("package-name", "", "Android application package name (e.g., com.example.app)")
	output := shared.BindOutputFlags(fs)

	return &ffcli.Command{
		Name:       "create",
		ShortUsage: "gplay edits create --package-name PACKAGE_NAME",
		ShortHelp:  "Create a new Google Play edit.",
		LongHelp: `Create a new Google Play edit.

Examples:
  gplay edits create --package-name "com.example.app"
  gplay edits create --package-name "com.example.app" --output json --pretty`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			packageNameValue, err := resolvePackageName(*packageName)
			if err != nil {
				return fmt.Errorf("googleplay edits create: %w", err)
			}
			if packageNameValue == "" {
				fmt.Fprintln(os.Stderr, "Error: --package-name is required (or set GOOGLE_PLAY_PACKAGE_NAME / gplay config set --package-name)")
				return flag.ErrHelp
			}

			client, err := newGooglePlayClient()
			if err != nil {
				return fmt.Errorf("googleplay edits create: %w", err)
			}

			requestCtx, cancel := shared.ContextWithTimeout(ctx)
			defer cancel()

			edit, err := client.CreateEdit(requestCtx, packageNameValue)
			if err != nil {
				return fmt.Errorf("googleplay edits create: %w", err)
			}

			return shared.PrintOutput(edit, *output.Output, *output.Pretty)
		},
	}
}

// GooglePlayEditsCommitCommand returns the edits commit subcommand.
func GooglePlayEditsCommitCommand() *ffcli.Command {
	fs := flag.NewFlagSet("edits commit", flag.ExitOnError)

	packageName := fs.String("package-name", "", "Android application package name (e.g., com.example.app)")
	editID := fs.String("edit-id", "", "Google Play edit ID")
	changesNotSentForReview := fs.Bool("changes-not-sent-for-review", false, "Commit changes without sending for review")
	output := shared.BindOutputFlags(fs)

	return &ffcli.Command{
		Name:       "commit",
		ShortUsage: "gplay edits commit --package-name PACKAGE_NAME --edit-id EDIT_ID [flags]",
		ShortHelp:  "Commit an existing Google Play edit.",
		LongHelp: `Commit an existing Google Play edit.

Examples:
  gplay edits commit --package-name "com.example.app" --edit-id "EDIT_ID"
  gplay edits commit --package-name "com.example.app" --edit-id "EDIT_ID" --changes-not-sent-for-review`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			packageNameValue, err := resolvePackageName(*packageName)
			if err != nil {
				return fmt.Errorf("googleplay edits commit: %w", err)
			}
			if packageNameValue == "" {
				fmt.Fprintln(os.Stderr, "Error: --package-name is required (or set GOOGLE_PLAY_PACKAGE_NAME / gplay config set --package-name)")
				return flag.ErrHelp
			}

			editIDValue := strings.TrimSpace(*editID)
			if editIDValue == "" {
				fmt.Fprintln(os.Stderr, "Error: --edit-id is required")
				return flag.ErrHelp
			}

			var changesFlag *bool
			fs.Visit(func(f *flag.Flag) {
				if f.Name == "changes-not-sent-for-review" {
					v := *changesNotSentForReview
					changesFlag = &v
				}
			})

			client, err := newGooglePlayClient()
			if err != nil {
				return fmt.Errorf("googleplay edits commit: %w", err)
			}

			requestCtx, cancel := shared.ContextWithTimeout(ctx)
			defer cancel()

			edit, err := client.CommitEdit(requestCtx, packageNameValue, editIDValue, changesFlag)
			if err != nil {
				return fmt.Errorf("googleplay edits commit: %w", err)
			}

			return shared.PrintOutput(edit, *output.Output, *output.Pretty)
		},
	}
}
