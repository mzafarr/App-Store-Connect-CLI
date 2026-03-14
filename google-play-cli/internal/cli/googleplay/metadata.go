package googleplay

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/rudrankriyam/google-play-cli/internal/cli/shared"
	play "github.com/rudrankriyam/google-play-cli/internal/googleplay"
)

type releaseNotesRelease struct {
	Status       string               `json:"status"`
	VersionCodes []int64              `json:"versionCodes,omitempty"`
	ReleaseNotes []play.LocalizedText `json:"releaseNotes,omitempty"`
}

type releaseNotesView struct {
	PackageName string                `json:"packageName"`
	Track       string                `json:"track"`
	Releases    []releaseNotesRelease `json:"releases"`
}

// GooglePlayMetadataCommand returns metadata command group.
func GooglePlayMetadataCommand() *ffcli.Command {
	fs := flag.NewFlagSet("metadata", flag.ExitOnError)

	return &ffcli.Command{
		Name:       "metadata",
		ShortUsage: "gplay metadata <subcommand> [flags]",
		ShortHelp:  "Manage release metadata such as localized changelogs.",
		LongHelp: `Manage release metadata such as localized changelogs.

Examples:
  gplay metadata release-notes get --package-name "com.example.app" --track "production"
  gplay metadata release-notes set --package-name "com.example.app" --edit-id "EDIT_ID" --track "production" --version-codes "123" --status completed --locale en-US --text "Bug fixes" --confirm-production`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			GooglePlayReleaseNotesCommand(),
		},
		Exec: func(ctx context.Context, args []string) error {
			return flag.ErrHelp
		},
	}
}

// GooglePlayReleaseNotesCommand returns release-notes command group.
func GooglePlayReleaseNotesCommand() *ffcli.Command {
	fs := flag.NewFlagSet("metadata release-notes", flag.ExitOnError)

	return &ffcli.Command{
		Name:       "release-notes",
		ShortUsage: "gplay metadata release-notes <subcommand> [flags]",
		ShortHelp:  "Get and set localized release notes for track releases.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			GooglePlayReleaseNotesGetCommand(),
			GooglePlayReleaseNotesSetCommand(),
		},
		Exec: func(ctx context.Context, args []string) error {
			return flag.ErrHelp
		},
	}
}

// GooglePlayReleaseNotesGetCommand fetches release notes from a track.
func GooglePlayReleaseNotesGetCommand() *ffcli.Command {
	fs := flag.NewFlagSet("metadata release-notes get", flag.ExitOnError)
	packageName := fs.String("package-name", "", "Android application package name (e.g., com.example.app)")
	track := fs.String("track", "", "Track name (e.g., internal, alpha, beta, production)")
	output := shared.BindOutputFlags(fs)

	return &ffcli.Command{
		Name:       "get",
		ShortUsage: "gplay metadata release-notes get --package-name PACKAGE_NAME --track TRACK",
		ShortHelp:  "Get localized release notes for a track.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			packageNameValue, err := resolvePackageName(*packageName)
			if err != nil {
				return fmt.Errorf("googleplay metadata release-notes get: %w", err)
			}
			if packageNameValue == "" {
				fmt.Fprintln(os.Stderr, "Error: --package-name is required (or set GOOGLE_PLAY_PACKAGE_NAME / gplay config set --package-name)")
				return flag.ErrHelp
			}
			trackValue := strings.TrimSpace(*track)
			if trackValue == "" {
				fmt.Fprintln(os.Stderr, "Error: --track is required")
				return flag.ErrHelp
			}

			client, err := newGooglePlayClient()
			if err != nil {
				return fmt.Errorf("googleplay metadata release-notes get: %w", err)
			}

			requestCtx, cancel := shared.ContextWithTimeout(ctx)
			defer cancel()

			edit, err := client.CreateEdit(requestCtx, packageNameValue)
			if err != nil {
				return fmt.Errorf("googleplay metadata release-notes get: %w", err)
			}

			trackState, err := client.GetTrack(requestCtx, packageNameValue, edit.ID, trackValue)
			if err != nil {
				return fmt.Errorf("googleplay metadata release-notes get: %w", err)
			}

			view := releaseNotesView{
				PackageName: packageNameValue,
				Track:       trackValue,
				Releases:    make([]releaseNotesRelease, 0, len(trackState.Releases)),
			}
			for _, release := range trackState.Releases {
				view.Releases = append(view.Releases, releaseNotesRelease{
					Status:       release.Status,
					VersionCodes: release.VersionCodes,
					ReleaseNotes: release.ReleaseNotes,
				})
			}

			return shared.PrintOutput(view, *output.Output, *output.Pretty)
		},
	}
}

// GooglePlayReleaseNotesSetCommand updates localized release notes on a track release.
func GooglePlayReleaseNotesSetCommand() *ffcli.Command {
	fs := flag.NewFlagSet("metadata release-notes set", flag.ExitOnError)
	packageName := fs.String("package-name", "", "Android application package name (e.g., com.example.app)")
	editID := fs.String("edit-id", "", "Google Play edit ID")
	track := fs.String("track", "", "Track name (e.g., internal, alpha, beta, production)")
	versionCodes := fs.String("version-codes", "", "Version codes to publish, comma-separated")
	status := fs.String("status", "", "Release status: draft, inProgress, halted, completed")
	locale := fs.String("locale", "", "Locale tag for release notes (e.g., en-US)")
	text := fs.String("text", "", "Localized release notes text")
	releaseName := fs.String("release-name", "", "Optional release name shown in Play Console")
	userFraction := fs.Float64("user-fraction", 0, "Fraction of users for staged rollout (required with --status inProgress, range >0 and <1)")
	inAppUpdatePriority := fs.Int("in-app-update-priority", 0, "In-app update priority (0-5)")
	confirmProduction := fs.Bool("confirm-production", false, "Required for production track writes")
	output := shared.BindOutputFlags(fs)

	return &ffcli.Command{
		Name:       "set",
		ShortUsage: "gplay metadata release-notes set --package-name PACKAGE_NAME --edit-id EDIT_ID --track TRACK --version-codes CODES --status STATUS --locale LOCALE --text TEXT",
		ShortHelp:  "Set localized release notes for a track release.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			packageNameValue, err := resolvePackageName(*packageName)
			if err != nil {
				return fmt.Errorf("googleplay metadata release-notes set: %w", err)
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

			localeValue := strings.TrimSpace(*locale)
			if localeValue == "" {
				fmt.Fprintln(os.Stderr, "Error: --locale is required")
				return flag.ErrHelp
			}
			textValue := strings.TrimSpace(*text)
			if textValue == "" {
				fmt.Fprintln(os.Stderr, "Error: --text is required")
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
				ReleaseNotes: []play.LocalizedText{
					{
						Language: localeValue,
						Text:     textValue,
					},
				},
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
				return fmt.Errorf("googleplay metadata release-notes set: %w", err)
			}

			requestCtx, cancel := shared.ContextWithTimeout(ctx)
			defer cancel()

			updatedTrack, err := client.UpdateTrack(
				requestCtx,
				packageNameValue,
				editIDValue,
				trackValue,
				play.UpdateTrackRequest{Releases: []play.TrackRelease{release}},
			)
			if err != nil {
				return fmt.Errorf("googleplay metadata release-notes set: %w", err)
			}

			return shared.PrintOutput(updatedTrack, *output.Output, *output.Pretty)
		},
	}
}
