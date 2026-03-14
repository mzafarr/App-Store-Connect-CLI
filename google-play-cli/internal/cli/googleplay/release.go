package googleplay

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/rudrankriyam/google-play-cli/internal/cli/shared"
	play "github.com/rudrankriyam/google-play-cli/internal/googleplay"
)

const (
	defaultPollInterval = 15 * time.Second
	defaultWaitTimeout  = 10 * time.Minute
)

// GooglePlayReleaseCommand returns the release command group.
func GooglePlayReleaseCommand() *ffcli.Command {
	fs := flag.NewFlagSet("release", flag.ExitOnError)

	return &ffcli.Command{
		Name:       "release",
		ShortUsage: "gplay release <subcommand> [flags]",
		ShortHelp:  "Run high-level Google Play release workflows.",
		LongHelp: `Run high-level Google Play release workflows.

Examples:
  gplay release run --package-name "com.example.app" --track "production" --status completed --aab "./app-release.aab"
  gplay release status --package-name "com.example.app" --track "production"
  gplay release status --package-name "com.example.app" --track "production" --version-codes "123" --wait`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			GooglePlayReleaseRunCommand(),
			GooglePlayReleaseStatusCommand(),
		},
		Exec: func(ctx context.Context, args []string) error {
			return flag.ErrHelp
		},
	}
}

type releaseRunOutput struct {
	PackageName string               `json:"packageName"`
	Track       string               `json:"track"`
	Status      string               `json:"status"`
	DryRun      bool                 `json:"dryRun"`
	Steps       []string             `json:"steps"`
	Edit        *play.AppEdit        `json:"edit,omitempty"`
	Bundle      *play.Bundle         `json:"bundle,omitempty"`
	APK         *play.APK            `json:"apk,omitempty"`
	TrackResult *play.Track          `json:"trackResult,omitempty"`
	Commit      *play.AppEdit        `json:"commit,omitempty"`
	StatusCheck *releaseStatusOutput `json:"statusCheck,omitempty"`
}

type observedRelease struct {
	Status       string  `json:"status"`
	VersionCodes []int64 `json:"versionCodes,omitempty"`
}

type releaseStatusOutput struct {
	PackageName     string            `json:"packageName"`
	Track           string            `json:"track"`
	UntilStatus     string            `json:"untilStatus"`
	Wait            bool              `json:"wait"`
	Ready           bool              `json:"ready"`
	VersionCodes    []int64           `json:"versionCodes,omitempty"`
	PollInterval    string            `json:"pollInterval,omitempty"`
	Elapsed         string            `json:"elapsed,omitempty"`
	ObservedRelease []observedRelease `json:"observedReleases,omitempty"`
	TrackState      *play.Track       `json:"trackState,omitempty"`
}

// GooglePlayReleaseRunCommand runs a complete release path.
func GooglePlayReleaseRunCommand() *ffcli.Command {
	fs := flag.NewFlagSet("release run", flag.ExitOnError)

	packageName := fs.String("package-name", "", "Android application package name (e.g., com.example.app)")
	track := fs.String("track", "", "Track name (e.g., internal, alpha, beta, production)")
	status := fs.String("status", "", "Release status: draft, inProgress, halted, completed")
	aabPath := fs.String("aab", "", "Path to .aab file to upload")
	apkPath := fs.String("apk", "", "Path to .apk file to upload")
	versionCodes := fs.String("version-codes", "", "Existing version codes to release, comma-separated")
	releaseName := fs.String("release-name", "", "Optional release name shown in Play Console")
	userFraction := fs.Float64("user-fraction", 0, "Fraction of users for staged rollout (required with --status inProgress, range >0 and <1)")
	inAppUpdatePriority := fs.Int("in-app-update-priority", 0, "In-app update priority (0-5)")
	changesNotSentForReview := fs.Bool("changes-not-sent-for-review", false, "Commit changes without sending for review")
	confirmProduction := fs.Bool("confirm-production", false, "Required for production track writes")
	wait := fs.Bool("wait", false, "Wait until release reaches --until-status")
	untilStatus := fs.String("until-status", "", "Target status when waiting: draft, inProgress, halted, completed (default: --status value)")
	pollInterval := fs.Duration("poll-interval", defaultPollInterval, "Polling interval when waiting")
	waitTimeout := fs.Duration("wait-timeout", defaultWaitTimeout, "Maximum wait duration when --wait is set")
	dryRun := fs.Bool("dry-run", false, "Validate inputs and print planned steps without making API calls")
	output := shared.BindOutputFlags(fs)

	return &ffcli.Command{
		Name:       "run",
		ShortUsage: "gplay release run --package-name PACKAGE_NAME --track TRACK --status STATUS (--aab PATH | --apk PATH | --version-codes CODES) [flags]",
		ShortHelp:  "Create edit, upload artifact/version, update track, and commit.",
		LongHelp: `Create edit, upload artifact/version, update track, and commit.

Exactly one of --aab, --apk, or --version-codes must be provided.

Examples:
  gplay release run --package-name "com.example.app" --track "production" --status completed --aab "./app-release.aab" --confirm-production
  gplay release run --package-name "com.example.app" --track "production" --status inProgress --apk "./app-release.apk" --user-fraction 0.1 --confirm-production
  gplay release run --package-name "com.example.app" --track "production" --status completed --version-codes "123" --wait --confirm-production`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			visited := map[string]bool{}
			fs.Visit(func(f *flag.Flag) {
				visited[f.Name] = true
			})

			packageNameValue, err := resolvePackageName(*packageName)
			if err != nil {
				return fmt.Errorf("googleplay release run: %w", err)
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

			statusValue, ok := normalizeReleaseStatus(*status)
			if !ok {
				fmt.Fprintln(os.Stderr, "Error: --status must be one of: draft, inProgress, halted, completed")
				return flag.ErrHelp
			}

			aabValue := strings.TrimSpace(*aabPath)
			apkValue := strings.TrimSpace(*apkPath)
			versionCodesValue := strings.TrimSpace(*versionCodes)
			selectedInputs := 0
			if aabValue != "" {
				selectedInputs++
			}
			if apkValue != "" {
				selectedInputs++
			}
			if versionCodesValue != "" {
				selectedInputs++
			}
			if selectedInputs == 0 {
				fmt.Fprintln(os.Stderr, "Error: exactly one of --aab, --apk, or --version-codes is required")
				return flag.ErrHelp
			}
			if selectedInputs > 1 {
				fmt.Fprintln(os.Stderr, "Error: --aab, --apk, and --version-codes are mutually exclusive")
				return flag.ErrHelp
			}

			if aabValue != "" {
				if !strings.EqualFold(filepath.Ext(aabValue), ".aab") {
					fmt.Fprintln(os.Stderr, "Error: --aab must point to a .aab file")
					return flag.ErrHelp
				}
				if _, err := os.Stat(aabValue); err != nil {
					fmt.Fprintf(os.Stderr, "Error: --aab file is not readable: %v\n", err)
					return flag.ErrHelp
				}
			}
			if apkValue != "" {
				if !strings.EqualFold(filepath.Ext(apkValue), ".apk") {
					fmt.Fprintln(os.Stderr, "Error: --apk must point to a .apk file")
					return flag.ErrHelp
				}
				if _, err := os.Stat(apkValue); err != nil {
					fmt.Fprintf(os.Stderr, "Error: --apk file is not readable: %v\n", err)
					return flag.ErrHelp
				}
			}

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

			waitStatus := statusValue
			if rawUntil := strings.TrimSpace(*untilStatus); rawUntil != "" {
				normalized, ok := normalizeReleaseStatus(rawUntil)
				if !ok {
					fmt.Fprintln(os.Stderr, "Error: --until-status must be one of: draft, inProgress, halted, completed")
					return flag.ErrHelp
				}
				waitStatus = normalized
			}
			if *wait {
				if *pollInterval <= 0 {
					fmt.Fprintln(os.Stderr, "Error: --poll-interval must be greater than 0")
					return flag.ErrHelp
				}
				if *waitTimeout <= 0 {
					fmt.Fprintln(os.Stderr, "Error: --wait-timeout must be greater than 0")
					return flag.ErrHelp
				}
			}

			var parsedVersionCodes []int64
			if versionCodesValue != "" {
				var err error
				parsedVersionCodes, err = parseVersionCodes(versionCodesValue)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error: %s\n", err)
					return flag.ErrHelp
				}
			}
			if !*dryRun {
				if err := requireProductionConfirmation(trackValue, *confirmProduction); err != nil {
					return err
				}
			}

			result := releaseRunOutput{
				PackageName: packageNameValue,
				Track:       trackValue,
				Status:      statusValue,
				DryRun:      *dryRun,
				Steps: []string{
					"create edit",
					"upload artifact or use provided version codes",
					"update track",
					"commit edit",
				},
			}
			if *wait {
				result.Steps = append(result.Steps, "wait for release status")
			}
			if *dryRun {
				statusOutput := releaseStatusOutput{
					PackageName:  packageNameValue,
					Track:        trackValue,
					UntilStatus:  waitStatus,
					Wait:         *wait,
					VersionCodes: parsedVersionCodes,
					PollInterval: pollInterval.String(),
				}
				if *wait {
					result.StatusCheck = &statusOutput
				}
				return shared.PrintOutput(result, *output.Output, *output.Pretty)
			}

			client, err := newGooglePlayClient()
			if err != nil {
				return fmt.Errorf("googleplay release run: %w", err)
			}

			requestCtx, cancel := shared.ContextWithTimeout(ctx)
			defer cancel()

			edit, err := client.CreateEdit(requestCtx, packageNameValue)
			if err != nil {
				return fmt.Errorf("googleplay release run: create edit: %w", err)
			}
			result.Edit = &edit

			if aabValue != "" {
				bundle, err := client.UploadBundle(requestCtx, packageNameValue, edit.ID, aabValue)
				if err != nil {
					return fmt.Errorf("googleplay release run: upload bundle: %w", err)
				}
				result.Bundle = &bundle
				versionCode := bundle.VersionCode.Int64()
				if versionCode <= 0 {
					return fmt.Errorf("googleplay release run: upload bundle response missing versionCode")
				}
				parsedVersionCodes = []int64{versionCode}
			}
			if apkValue != "" {
				apk, err := client.UploadAPK(requestCtx, packageNameValue, edit.ID, apkValue)
				if err != nil {
					return fmt.Errorf("googleplay release run: upload apk: %w", err)
				}
				result.APK = &apk
				versionCode := apk.VersionCode.Int64()
				if versionCode <= 0 {
					return fmt.Errorf("googleplay release run: upload apk response missing versionCode")
				}
				parsedVersionCodes = []int64{versionCode}
			}

			release := play.TrackRelease{
				Name:         strings.TrimSpace(*releaseName),
				Status:       statusValue,
				VersionCodes: parsedVersionCodes,
			}
			if visited["user-fraction"] {
				v := *userFraction
				release.UserFraction = &v
			}
			if visited["in-app-update-priority"] {
				p := *inAppUpdatePriority
				release.InAppUpdatePriority = &p
			}

			updatedTrack, err := client.UpdateTrack(
				requestCtx,
				packageNameValue,
				edit.ID,
				trackValue,
				play.UpdateTrackRequest{Releases: []play.TrackRelease{release}},
			)
			if err != nil {
				return fmt.Errorf("googleplay release run: update track: %w", err)
			}
			result.TrackResult = &updatedTrack

			var commitFlag *bool
			if visited["changes-not-sent-for-review"] {
				v := *changesNotSentForReview
				commitFlag = &v
			}
			commit, err := client.CommitEdit(requestCtx, packageNameValue, edit.ID, commitFlag)
			if err != nil {
				return fmt.Errorf("googleplay release run: commit edit: %w", err)
			}
			result.Commit = &commit

			if *wait {
				waitCtx, waitCancel := context.WithTimeout(ctx, *waitTimeout)
				defer waitCancel()

				statusResult, err := waitForReleaseStatus(waitCtx, client, packageNameValue, trackValue, parsedVersionCodes, waitStatus, *pollInterval, true)
				if err != nil {
					return fmt.Errorf("googleplay release run: wait status: %w", err)
				}
				result.StatusCheck = &statusResult
			}

			return shared.PrintOutput(result, *output.Output, *output.Pretty)
		},
	}
}

// GooglePlayReleaseStatusCommand fetches release state and optionally waits.
func GooglePlayReleaseStatusCommand() *ffcli.Command {
	fs := flag.NewFlagSet("release status", flag.ExitOnError)

	packageName := fs.String("package-name", "", "Android application package name (e.g., com.example.app)")
	track := fs.String("track", "", "Track name (e.g., internal, alpha, beta, production)")
	versionCodes := fs.String("version-codes", "", "Version codes to check, comma-separated")
	untilStatus := fs.String("until-status", "completed", "Target status: draft, inProgress, halted, completed")
	wait := fs.Bool("wait", false, "Poll until target status is reached")
	pollInterval := fs.Duration("poll-interval", defaultPollInterval, "Polling interval when waiting")
	waitTimeout := fs.Duration("wait-timeout", defaultWaitTimeout, "Maximum wait duration when --wait is set")
	output := shared.BindOutputFlags(fs)

	return &ffcli.Command{
		Name:       "status",
		ShortUsage: "gplay release status --package-name PACKAGE_NAME --track TRACK [flags]",
		ShortHelp:  "Get release status for a track, with optional waiting.",
		LongHelp: `Get release status for a track, with optional waiting.

Examples:
  gplay release status --package-name "com.example.app" --track "production"
  gplay release status --package-name "com.example.app" --track "production" --version-codes "123" --wait --until-status completed`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			packageNameValue, err := resolvePackageName(*packageName)
			if err != nil {
				return fmt.Errorf("googleplay release status: %w", err)
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

			untilStatusValue, ok := normalizeReleaseStatus(*untilStatus)
			if !ok {
				fmt.Fprintln(os.Stderr, "Error: --until-status must be one of: draft, inProgress, halted, completed")
				return flag.ErrHelp
			}

			versionCodeValues, err := parseVersionCodesAllowEmpty(*versionCodes)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %s\n", err)
				return flag.ErrHelp
			}

			if *wait {
				if *pollInterval <= 0 {
					fmt.Fprintln(os.Stderr, "Error: --poll-interval must be greater than 0")
					return flag.ErrHelp
				}
				if *waitTimeout <= 0 {
					fmt.Fprintln(os.Stderr, "Error: --wait-timeout must be greater than 0")
					return flag.ErrHelp
				}
			}

			client, err := newGooglePlayClient()
			if err != nil {
				return fmt.Errorf("googleplay release status: %w", err)
			}

			if *wait {
				waitCtx, cancel := context.WithTimeout(ctx, *waitTimeout)
				defer cancel()

				result, err := waitForReleaseStatus(waitCtx, client, packageNameValue, trackValue, versionCodeValues, untilStatusValue, *pollInterval, true)
				if err != nil {
					return fmt.Errorf("googleplay release status: %w", err)
				}
				return shared.PrintOutput(result, *output.Output, *output.Pretty)
			}

			result, err := waitForReleaseStatus(ctx, client, packageNameValue, trackValue, versionCodeValues, untilStatusValue, *pollInterval, false)
			if err != nil {
				return fmt.Errorf("googleplay release status: %w", err)
			}
			return shared.PrintOutput(result, *output.Output, *output.Pretty)
		},
	}
}

func waitForReleaseStatus(
	ctx context.Context,
	client *play.Client,
	packageName string,
	track string,
	versionCodes []int64,
	untilStatus string,
	pollInterval time.Duration,
	wait bool,
) (releaseStatusOutput, error) {
	start := time.Now()
	last := releaseStatusOutput{
		PackageName:  packageName,
		Track:        track,
		UntilStatus:  untilStatus,
		Wait:         wait,
		VersionCodes: versionCodes,
		PollInterval: pollInterval.String(),
	}

	read := func() (releaseStatusOutput, error) {
		requestCtx, cancel := shared.ContextWithTimeout(ctx)
		defer cancel()

		edit, err := client.CreateEdit(requestCtx, packageName)
		if err != nil {
			return releaseStatusOutput{}, err
		}

		trackState, err := client.GetTrack(requestCtx, packageName, edit.ID, track)
		if err != nil {
			return releaseStatusOutput{}, err
		}

		ready, observed := evaluateTrackReady(trackState, versionCodes, untilStatus)
		result := releaseStatusOutput{
			PackageName:     packageName,
			Track:           track,
			UntilStatus:     untilStatus,
			Wait:            wait,
			Ready:           ready,
			VersionCodes:    versionCodes,
			PollInterval:    pollInterval.String(),
			ObservedRelease: observed,
			TrackState:      &trackState,
		}
		return result, nil
	}

	snapshot, err := read()
	if err != nil {
		return releaseStatusOutput{}, err
	}
	snapshot.Elapsed = time.Since(start).Round(time.Second).String()
	last = snapshot
	if !wait || snapshot.Ready {
		return snapshot, nil
	}

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			last.Elapsed = time.Since(start).Round(time.Second).String()
			return last, fmt.Errorf("timed out waiting for %q status on track %q", untilStatus, track)
		case <-ticker.C:
			snapshot, err := read()
			if err != nil {
				return releaseStatusOutput{}, err
			}
			snapshot.Elapsed = time.Since(start).Round(time.Second).String()
			last = snapshot
			if snapshot.Ready {
				return snapshot, nil
			}
		}
	}
}

func evaluateTrackReady(track play.Track, targetVersionCodes []int64, untilStatus string) (bool, []observedRelease) {
	observed := make([]observedRelease, 0, len(track.Releases))
	for _, rel := range track.Releases {
		observed = append(observed, observedRelease{
			Status:       strings.TrimSpace(rel.Status),
			VersionCodes: rel.VersionCodes,
		})
	}

	if len(track.Releases) == 0 {
		return false, observed
	}

	if len(targetVersionCodes) == 0 {
		for _, rel := range track.Releases {
			if strings.EqualFold(strings.TrimSpace(rel.Status), untilStatus) {
				return true, observed
			}
		}
		return false, observed
	}

	covered := map[int64]bool{}
	for _, rel := range track.Releases {
		for _, code := range rel.VersionCodes {
			for _, target := range targetVersionCodes {
				if code == target {
					if strings.EqualFold(strings.TrimSpace(rel.Status), untilStatus) {
						covered[target] = true
					}
				}
			}
		}
	}

	for _, target := range targetVersionCodes {
		if !covered[target] {
			return false, observed
		}
	}
	return true, observed
}

func parseVersionCodesAllowEmpty(raw string) ([]int64, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	return parseVersionCodes(raw)
}
