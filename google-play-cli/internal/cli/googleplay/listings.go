package googleplay

import (
	"context"
	"flag"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/rudrankriyam/google-play-cli/internal/cli/shared"
	play "github.com/rudrankriyam/google-play-cli/internal/googleplay"
)

var listingLanguagePattern = regexp.MustCompile(`^[A-Za-z]{2,3}(?:-[A-Za-z0-9]{2,8})*$`)

// GooglePlayListingsCommand returns listings command group.
func GooglePlayListingsCommand() *ffcli.Command {
	fs := flag.NewFlagSet("listings", flag.ExitOnError)

	return &ffcli.Command{
		Name:       "listings",
		ShortUsage: "gplay listings <subcommand> [flags]",
		ShortHelp:  "List, get, and update localized store listings.",
		LongHelp: `List, get, and update localized store listings.

Examples:
  gplay listings list --package-name "com.example.app" --edit-id "EDIT_ID"
  gplay listings get --package-name "com.example.app" --edit-id "EDIT_ID" --language en-US
  gplay listings update --package-name "com.example.app" --edit-id "EDIT_ID" --language en-US --title "New title"`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			GooglePlayListingsListCommand(),
			GooglePlayListingsGetCommand(),
			GooglePlayListingsUpdateCommand(),
		},
		Exec: func(ctx context.Context, args []string) error {
			return flag.ErrHelp
		},
	}
}

// GooglePlayListingsListCommand lists all localized store listings for an edit.
func GooglePlayListingsListCommand() *ffcli.Command {
	fs := flag.NewFlagSet("listings list", flag.ExitOnError)
	packageName := fs.String("package-name", "", "Android application package name (e.g., com.example.app)")
	editID := fs.String("edit-id", "", "Google Play edit ID")
	output := shared.BindOutputFlags(fs)

	return &ffcli.Command{
		Name:       "list",
		ShortUsage: "gplay listings list --package-name PACKAGE_NAME --edit-id EDIT_ID",
		ShortHelp:  "List localized store listings for an edit.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			packageNameValue, err := resolvePackageName(*packageName)
			if err != nil {
				return fmt.Errorf("googleplay listings list: %w", err)
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
				return fmt.Errorf("googleplay listings list: %w", err)
			}

			requestCtx, cancel := shared.ContextWithTimeout(ctx)
			defer cancel()

			listings, err := client.ListListings(requestCtx, packageNameValue, editIDValue)
			if err != nil {
				return fmt.Errorf("googleplay listings list: %w", err)
			}
			return shared.PrintOutput(listings, *output.Output, *output.Pretty)
		},
	}
}

// GooglePlayListingsGetCommand gets a specific localized listing by language.
func GooglePlayListingsGetCommand() *ffcli.Command {
	fs := flag.NewFlagSet("listings get", flag.ExitOnError)
	packageName := fs.String("package-name", "", "Android application package name (e.g., com.example.app)")
	editID := fs.String("edit-id", "", "Google Play edit ID")
	language := fs.String("language", "", "Language localization code (BCP-47, e.g., en-US)")
	output := shared.BindOutputFlags(fs)

	return &ffcli.Command{
		Name:       "get",
		ShortUsage: "gplay listings get --package-name PACKAGE_NAME --edit-id EDIT_ID --language LANGUAGE",
		ShortHelp:  "Get one localized store listing.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			packageNameValue, err := resolvePackageName(*packageName)
			if err != nil {
				return fmt.Errorf("googleplay listings get: %w", err)
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
			languageValue := strings.TrimSpace(*language)
			if languageValue == "" {
				fmt.Fprintln(os.Stderr, "Error: --language is required")
				return flag.ErrHelp
			}
			if !isValidListingLanguage(languageValue) {
				fmt.Fprintln(os.Stderr, "Error: --language must be a valid BCP-47 tag")
				return flag.ErrHelp
			}

			client, err := newGooglePlayClient()
			if err != nil {
				return fmt.Errorf("googleplay listings get: %w", err)
			}

			requestCtx, cancel := shared.ContextWithTimeout(ctx)
			defer cancel()

			listing, err := client.GetListing(requestCtx, packageNameValue, editIDValue, languageValue)
			if err != nil {
				return fmt.Errorf("googleplay listings get: %w", err)
			}
			return shared.PrintOutput(listing, *output.Output, *output.Pretty)
		},
	}
}

// GooglePlayListingsUpdateCommand patches a localized listing.
func GooglePlayListingsUpdateCommand() *ffcli.Command {
	fs := flag.NewFlagSet("listings update", flag.ExitOnError)
	packageName := fs.String("package-name", "", "Android application package name (e.g., com.example.app)")
	editID := fs.String("edit-id", "", "Google Play edit ID")
	language := fs.String("language", "", "Language localization code (BCP-47, e.g., en-US)")
	title := fs.String("title", "", "Localized app title")
	shortDescription := fs.String("short-description", "", "Localized short description")
	fullDescription := fs.String("full-description", "", "Localized full description")
	video := fs.String("video", "", "Promotional YouTube URL")
	output := shared.BindOutputFlags(fs)

	return &ffcli.Command{
		Name:       "update",
		ShortUsage: "gplay listings update --package-name PACKAGE_NAME --edit-id EDIT_ID --language LANGUAGE [content flags]",
		ShortHelp:  "Patch a localized store listing.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			packageNameValue, err := resolvePackageName(*packageName)
			if err != nil {
				return fmt.Errorf("googleplay listings update: %w", err)
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
			languageValue := strings.TrimSpace(*language)
			if languageValue == "" {
				fmt.Fprintln(os.Stderr, "Error: --language is required")
				return flag.ErrHelp
			}
			if !isValidListingLanguage(languageValue) {
				fmt.Fprintln(os.Stderr, "Error: --language must be a valid BCP-47 tag")
				return flag.ErrHelp
			}

			visited := map[string]bool{}
			fs.Visit(func(f *flag.Flag) {
				visited[f.Name] = true
			})

			if !visited["title"] && !visited["short-description"] && !visited["full-description"] && !visited["video"] {
				fmt.Fprintln(os.Stderr, "Error: at least one of --title, --short-description, --full-description, or --video is required")
				return flag.ErrHelp
			}

			videoValue := strings.TrimSpace(*video)
			if visited["video"] && videoValue != "" {
				videoURL, err := url.Parse(videoValue)
				if err != nil || videoURL.Scheme == "" || videoURL.Host == "" {
					fmt.Fprintln(os.Stderr, "Error: --video must be a valid absolute URL")
					return flag.ErrHelp
				}
			}

			payload := play.Listing{
				Language: languageValue,
			}
			if visited["title"] {
				payload.Title = *title
			}
			if visited["short-description"] {
				payload.ShortDescription = *shortDescription
			}
			if visited["full-description"] {
				payload.FullDescription = *fullDescription
			}
			if visited["video"] {
				payload.Video = *video
			}

			client, err := newGooglePlayClient()
			if err != nil {
				return fmt.Errorf("googleplay listings update: %w", err)
			}

			requestCtx, cancel := shared.ContextWithTimeout(ctx)
			defer cancel()

			updatedListing, err := client.PatchListing(requestCtx, packageNameValue, editIDValue, languageValue, payload)
			if err != nil {
				return fmt.Errorf("googleplay listings update: %w", err)
			}
			return shared.PrintOutput(updatedListing, *output.Output, *output.Pretty)
		},
	}
}

func isValidListingLanguage(value string) bool {
	return listingLanguagePattern.MatchString(strings.TrimSpace(value))
}
