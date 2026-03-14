package googleplay

import (
	"context"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/rudrankriyam/google-play-cli/internal/cli/shared"
	play "github.com/rudrankriyam/google-play-cli/internal/googleplay"
)

type appsListOutput struct {
	Developer      string   `json:"developer"`
	UsersScanned   int      `json:"usersScanned"`
	PartialUsers   int      `json:"partialUsers,omitempty"`
	PackageNames   []string `json:"packageNames"`
	DraftGrantRefs []string `json:"draftGrantRefs,omitempty"`
}

// GooglePlayAppsCommand returns apps command group.
func GooglePlayAppsCommand() *ffcli.Command {
	fs := flag.NewFlagSet("apps", flag.ExitOnError)

	return &ffcli.Command{
		Name:       "apps",
		ShortUsage: "gplay apps <subcommand> [flags]",
		ShortHelp:  "Discover accessible apps for a developer account.",
		LongHelp: `Discover accessible apps for a developer account.

Examples:
  gplay apps list --developer 1234567890123456789
  gplay apps list --developer developers/1234567890123456789 --pretty`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			GooglePlayAppsListCommand(),
		},
		Exec: func(ctx context.Context, args []string) error {
			return flag.ErrHelp
		},
	}
}

// GooglePlayAppsListCommand lists package names visible through user grants.
func GooglePlayAppsListCommand() *ffcli.Command {
	fs := flag.NewFlagSet("apps list", flag.ExitOnError)
	developer := fs.String("developer", "", "Developer account ID or resource (e.g., 123... or developers/123...)")
	pageSize := fs.Int("page-size", -1, "Users page size; use -1 to disable pagination per API recommendation")
	includeDrafts := fs.Bool("include-drafts", false, "Include draft grant resource names when package name is empty")
	output := shared.BindOutputFlags(fs)

	return &ffcli.Command{
		Name:       "list",
		ShortUsage: "gplay apps list [--developer DEVELOPER_ID] [--page-size -1]",
		ShortHelp:  "List package names accessible in a developer account.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			developerValue, err := resolveDeveloper(*developer)
			if err != nil {
				return fmt.Errorf("googleplay apps list: %w", err)
			}
			if developerValue == "" {
				fmt.Fprintln(os.Stderr, "Error: --developer is required (or set GOOGLE_PLAY_DEVELOPER / gplay config set --developer)")
				return flag.ErrHelp
			}
			if *pageSize == 0 || *pageSize < -1 {
				fmt.Fprintln(os.Stderr, "Error: --page-size must be -1 or a positive integer")
				return flag.ErrHelp
			}

			client, err := newGooglePlayClient()
			if err != nil {
				return fmt.Errorf("googleplay apps list: %w", err)
			}

			users := make([]play.User, 0)
			pageToken := ""
			for {
				requestCtx, cancel := shared.ContextWithTimeout(ctx)
				resp, err := client.ListUsers(requestCtx, developerValue, *pageSize, pageToken)
				cancel()
				if err != nil {
					return fmt.Errorf("googleplay apps list: %w", err)
				}
				users = append(users, resp.Users...)

				pageToken = strings.TrimSpace(resp.NextPageToken)
				if pageToken == "" {
					break
				}
			}

			packages := map[string]struct{}{}
			draftRefs := map[string]struct{}{}
			partialUsers := 0

			for _, user := range users {
				if user.Partial {
					partialUsers++
				}
				for _, grant := range user.Grants {
					packageName := strings.TrimSpace(grant.PackageName)
					if packageName != "" {
						packages[packageName] = struct{}{}
						continue
					}
					if *includeDrafts {
						if grantRef := strings.TrimSpace(grant.Name); grantRef != "" {
							draftRefs[grantRef] = struct{}{}
						}
					}
				}
			}

			packageNames := make([]string, 0, len(packages))
			for packageName := range packages {
				packageNames = append(packageNames, packageName)
			}
			sort.Strings(packageNames)

			draftGrantRefs := make([]string, 0, len(draftRefs))
			for grantRef := range draftRefs {
				draftGrantRefs = append(draftGrantRefs, grantRef)
			}
			sort.Strings(draftGrantRefs)

			outputValue := appsListOutput{
				Developer:      normalizeDeveloperValue(developerValue),
				UsersScanned:   len(users),
				PartialUsers:   partialUsers,
				PackageNames:   packageNames,
				DraftGrantRefs: draftGrantRefs,
			}

			return shared.PrintOutput(outputValue, *output.Output, *output.Pretty)
		},
	}
}

func normalizeDeveloperValue(value string) string {
	if strings.HasPrefix(value, "developers/") {
		return value
	}
	return "developers/" + value
}
