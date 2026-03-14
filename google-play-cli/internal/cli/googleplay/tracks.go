package googleplay

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/rudrankriyam/google-play-cli/internal/cli/shared"
	play "github.com/rudrankriyam/google-play-cli/internal/googleplay"
)

// GooglePlayTracksCommand returns the tracks command group.
func GooglePlayTracksCommand() *ffcli.Command {
	fs := flag.NewFlagSet("tracks", flag.ExitOnError)

	return &ffcli.Command{
		Name:       "tracks",
		ShortUsage: "gplay tracks <subcommand> [flags]",
		ShortHelp:  "List and update Google Play release tracks.",
		LongHelp: `List and update Google Play release tracks.

Examples:
  gplay tracks list --package-name "com.example.app" --edit-id "EDIT_ID"
  gplay tracks update --package-name "com.example.app" --edit-id "EDIT_ID" --track "production" --version-codes "123" --status completed`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			GooglePlayTracksListCommand(),
			GooglePlayTracksUpdateCommand(),
		},
		Exec: func(ctx context.Context, args []string) error {
			return flag.ErrHelp
		},
	}
}

// GooglePlayTracksListCommand returns the tracks list subcommand.
func GooglePlayTracksListCommand() *ffcli.Command {
	fs := flag.NewFlagSet("tracks list", flag.ExitOnError)

	packageName := fs.String("package-name", "", "Android application package name (e.g., com.example.app)")
	editID := fs.String("edit-id", "", "Google Play edit ID")
	output := shared.BindOutputFlags(fs)

	return &ffcli.Command{
		Name:       "list",
		ShortUsage: "gplay tracks list --package-name PACKAGE_NAME --edit-id EDIT_ID",
		ShortHelp:  "List tracks for a Google Play edit.",
		LongHelp: `List tracks for a Google Play edit.

Examples:
  gplay tracks list --package-name "com.example.app" --edit-id "EDIT_ID"
  gplay tracks list --package-name "com.example.app" --edit-id "EDIT_ID" --output json --pretty`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			packageNameValue, err := resolvePackageName(*packageName)
			if err != nil {
				return fmt.Errorf("googleplay tracks list: %w", err)
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

			client, err := newGooglePlayClient()
			if err != nil {
				return fmt.Errorf("googleplay tracks list: %w", err)
			}

			requestCtx, cancel := shared.ContextWithTimeout(ctx)
			defer cancel()

			tracks, err := client.ListTracks(requestCtx, packageNameValue, editIDValue)
			if err != nil {
				return fmt.Errorf("googleplay tracks list: %w", err)
			}

			return shared.PrintOutput(tracks, *output.Output, *output.Pretty)
		},
	}
}

// GooglePlayTracksUpdateCommand returns the tracks update subcommand.
func GooglePlayTracksUpdateCommand() *ffcli.Command {
	fs := flag.NewFlagSet("tracks update", flag.ExitOnError)

	packageName := fs.String("package-name", "", "Android application package name (e.g., com.example.app)")
	editID := fs.String("edit-id", "", "Google Play edit ID")
	track := fs.String("track", "", "Track name (e.g., internal, alpha, beta, production)")
	versionCodes := fs.String("version-codes", "", "Version codes to publish, comma-separated (e.g., 123,124)")
	status := fs.String("status", "", "Release status: draft, inProgress, halted, completed")
	releaseName := fs.String("release-name", "", "Optional release name shown in Play Console")
	userFraction := fs.Float64("user-fraction", 0, "Fraction of users for staged rollout (required with --status inProgress, range >0 and <1)")
	inAppUpdatePriority := fs.Int("in-app-update-priority", 0, "In-app update priority (0-5)")
	confirmProduction := fs.Bool("confirm-production", false, "Required for production track writes")
	output := shared.BindOutputFlags(fs)

	return &ffcli.Command{
		Name:       "update",
		ShortUsage: "gplay tracks update --package-name PACKAGE_NAME --edit-id EDIT_ID --track TRACK --version-codes CODES --status STATUS [flags]",
		ShortHelp:  "Update releases for a Google Play track.",
		LongHelp: `Update releases for a Google Play track.

Examples:
  gplay tracks update --package-name "com.example.app" --edit-id "EDIT_ID" --track "production" --version-codes "123" --status completed --confirm-production
  gplay tracks update --package-name "com.example.app" --edit-id "EDIT_ID" --track "production" --version-codes "124" --status inProgress --user-fraction 0.1 --confirm-production`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			packageNameValue, err := resolvePackageName(*packageName)
			if err != nil {
				return fmt.Errorf("googleplay tracks update: %w", err)
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

			trackValue := strings.TrimSpace(*track)
			if trackValue == "" {
				fmt.Fprintln(os.Stderr, "Error: --track is required")
				return flag.ErrHelp
			}

			versionCodeValues, err := parseVersionCodes(*versionCodes)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %s\n", err)
				return flag.ErrHelp
			}

			statusValue, ok := normalizeReleaseStatus(*status)
			if !ok {
				fmt.Fprintln(os.Stderr, "Error: --status must be one of: draft, inProgress, halted, completed")
				return flag.ErrHelp
			}

			visited := map[string]bool{}
			fs.Visit(func(f *flag.Flag) {
				visited[f.Name] = true
			})

			if statusValue == "inProgress" {
				if !visited["user-fraction"] {
					fmt.Fprintln(os.Stderr, "Error: --user-fraction is required when --status is inProgress")
					return flag.ErrHelp
				}
				if *userFraction <= 0 || *userFraction >= 1 {
					fmt.Fprintln(os.Stderr, "Error: --user-fraction must be greater than 0 and less than 1")
					return flag.ErrHelp
				}
			} else if visited["user-fraction"] {
				fmt.Fprintln(os.Stderr, "Error: --user-fraction can only be used when --status is inProgress")
				return flag.ErrHelp
			}

			if visited["in-app-update-priority"] && (*inAppUpdatePriority < 0 || *inAppUpdatePriority > 5) {
				fmt.Fprintln(os.Stderr, "Error: --in-app-update-priority must be between 0 and 5")
				return flag.ErrHelp
			}
			if err := requireProductionConfirmation(trackValue, *confirmProduction); err != nil {
				return err
			}

			release := play.TrackRelease{
				Name:         strings.TrimSpace(*releaseName),
				Status:       statusValue,
				VersionCodes: versionCodeValues,
			}
			if visited["user-fraction"] {
				v := *userFraction
				release.UserFraction = &v
			}
			if visited["in-app-update-priority"] {
				p := *inAppUpdatePriority
				release.InAppUpdatePriority = &p
			}

			client, err := newGooglePlayClient()
			if err != nil {
				return fmt.Errorf("googleplay tracks update: %w", err)
			}

			requestCtx, cancel := shared.ContextWithTimeout(ctx)
			defer cancel()

			updatedTrack, err := client.UpdateTrack(
				requestCtx,
				packageNameValue,
				editIDValue,
				trackValue,
				play.UpdateTrackRequest{
					Releases: []play.TrackRelease{release},
				},
			)
			if err != nil {
				return fmt.Errorf("googleplay tracks update: %w", err)
			}

			return shared.PrintOutput(updatedTrack, *output.Output, *output.Pretty)
		},
	}
}

func parseVersionCodes(raw string) ([]int64, error) {
	values := shared.SplitCSV(raw)
	if len(values) == 0 {
		return nil, fmt.Errorf("--version-codes is required")
	}

	versionCodes := make([]int64, 0, len(values))
	for _, value := range values {
		versionCode, err := strconv.ParseInt(value, 10, 64)
		if err != nil || versionCode <= 0 {
			return nil, fmt.Errorf("--version-codes must contain positive integers")
		}
		versionCodes = append(versionCodes, versionCode)
	}

	return versionCodes, nil
}

func normalizeReleaseStatus(raw string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "draft":
		return "draft", true
	case "inprogress", "in-progress":
		return "inProgress", true
	case "halted":
		return "halted", true
	case "completed", "complete":
		return "completed", true
	default:
		return "", false
	}
}
