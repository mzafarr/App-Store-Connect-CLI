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

// GooglePlayAPKsCommand returns the APK command group.
func GooglePlayAPKsCommand() *ffcli.Command {
	fs := flag.NewFlagSet("apks", flag.ExitOnError)

	return &ffcli.Command{
		Name:       "apks",
		ShortUsage: "gplay apks <subcommand> [flags]",
		ShortHelp:  "Upload APKs to Google Play edits.",
		LongHelp: `Upload APKs to Google Play edits.

Examples:
  gplay apks upload --package-name "com.example.app" --edit-id "EDIT_ID" --apk "./app-release.apk"`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			GooglePlayAPKsUploadCommand(),
		},
		Exec: func(ctx context.Context, args []string) error {
			return flag.ErrHelp
		},
	}
}

// GooglePlayAPKsUploadCommand uploads an APK to a Google Play edit.
func GooglePlayAPKsUploadCommand() *ffcli.Command {
	fs := flag.NewFlagSet("apks upload", flag.ExitOnError)

	packageName := fs.String("package-name", "", "Android application package name (e.g., com.example.app)")
	editID := fs.String("edit-id", "", "Google Play edit ID")
	apkPath := fs.String("apk", "", "Path to .apk file")
	output := shared.BindOutputFlags(fs)

	return &ffcli.Command{
		Name:       "upload",
		ShortUsage: "gplay apks upload --package-name PACKAGE_NAME --edit-id EDIT_ID --apk PATH",
		ShortHelp:  "Upload an APK file to an edit.",
		LongHelp: `Upload an APK file to an edit.

Examples:
  gplay apks upload --package-name "com.example.app" --edit-id "EDIT_ID" --apk "./app-release.apk"
  gplay apks upload --package-name "com.example.app" --edit-id "EDIT_ID" --apk "./app-release.apk" --output json --pretty`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			packageNameValue, err := resolvePackageName(*packageName)
			if err != nil {
				return fmt.Errorf("googleplay apks upload: %w", err)
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

			apkValue := strings.TrimSpace(*apkPath)
			if apkValue == "" {
				fmt.Fprintln(os.Stderr, "Error: --apk is required")
				return flag.ErrHelp
			}
			if !strings.EqualFold(filepath.Ext(apkValue), ".apk") {
				fmt.Fprintln(os.Stderr, "Error: --apk must point to a .apk file")
				return flag.ErrHelp
			}

			client, err := newGooglePlayClient()
			if err != nil {
				return fmt.Errorf("googleplay apks upload: %w", err)
			}

			requestCtx, cancel := shared.ContextWithTimeout(ctx)
			defer cancel()

			apk, err := client.UploadAPK(requestCtx, packageNameValue, editIDValue, apkValue)
			if err != nil {
				return fmt.Errorf("googleplay apks upload: %w", err)
			}

			return shared.PrintOutput(apk, *output.Output, *output.Pretty)
		},
	}
}
