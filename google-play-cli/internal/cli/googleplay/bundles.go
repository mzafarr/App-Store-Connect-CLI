package googleplay

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/rudrankriyam/google-play-cli/internal/cli/shared"
)

// GooglePlayBundlesCommand returns the bundles command group.
func GooglePlayBundlesCommand() *ffcli.Command {
	fs := flag.NewFlagSet("bundles", flag.ExitOnError)

	return &ffcli.Command{
		Name:       "bundles",
		ShortUsage: "gplay bundles <subcommand> [flags]",
		ShortHelp:  "Upload Android App Bundles to Google Play edits.",
		LongHelp: `Upload Android App Bundles to Google Play edits.

Examples:
  gplay bundles upload --package-name "com.example.app" --edit-id "EDIT_ID" --aab "./app-release.aab"`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			GooglePlayBundlesUploadCommand(),
		},
		Exec: func(ctx context.Context, args []string) error {
			return flag.ErrHelp
		},
	}
}

// GooglePlayBundlesUploadCommand uploads an AAB to a Google Play edit.
func GooglePlayBundlesUploadCommand() *ffcli.Command {
	fs := flag.NewFlagSet("bundles upload", flag.ExitOnError)

	packageName := fs.String("package-name", "", "Android application package name (e.g., com.example.app)")
	editID := fs.String("edit-id", "", "Google Play edit ID")
	aabPath := fs.String("aab", "", "Path to .aab file")
	output := shared.BindOutputFlags(fs)

	return &ffcli.Command{
		Name:       "upload",
		ShortUsage: "gplay bundles upload --package-name PACKAGE_NAME --edit-id EDIT_ID --aab PATH",
		ShortHelp:  "Upload an Android App Bundle (.aab) to an edit.",
		LongHelp: `Upload an Android App Bundle (.aab) to an edit.

Examples:
  gplay bundles upload --package-name "com.example.app" --edit-id "EDIT_ID" --aab "./app-release.aab"
  gplay bundles upload --package-name "com.example.app" --edit-id "EDIT_ID" --aab "./app-release.aab" --output json --pretty`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			packageNameValue, err := resolvePackageName(*packageName)
			if err != nil {
				return fmt.Errorf("googleplay bundles upload: %w", err)
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

			aabValue := strings.TrimSpace(*aabPath)
			if aabValue == "" {
				fmt.Fprintln(os.Stderr, "Error: --aab is required")
				return flag.ErrHelp
			}
			if !strings.EqualFold(filepath.Ext(aabValue), ".aab") {
				fmt.Fprintln(os.Stderr, "Error: --aab must point to a .aab file")
				return flag.ErrHelp
			}

			client, err := newGooglePlayClient()
			if err != nil {
				return fmt.Errorf("googleplay bundles upload: %w", err)
			}

			requestCtx, cancel := shared.ContextWithTimeout(ctx)
			defer cancel()

			bundle, err := client.UploadBundle(requestCtx, packageNameValue, editIDValue, aabValue)
			if err != nil {
				return fmt.Errorf("googleplay bundles upload: %w", err)
			}

			return shared.PrintOutput(bundle, *output.Output, *output.Pretty)
		},
	}
}
