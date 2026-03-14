package googleplay

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/rudrankriyam/google-play-cli/internal/cli/shared"
	"github.com/rudrankriyam/google-play-cli/internal/config"
)

type configShowOutput struct {
	Path                string         `json:"path"`
	Exists              bool           `json:"exists"`
	SelectedProfile     string         `json:"selectedProfile,omitempty"`
	ResolvedDeveloper   string         `json:"resolvedDeveloper,omitempty"`
	ResolvedPackageName string         `json:"resolvedPackageName,omitempty"`
	Config              *config.Config `json:"config,omitempty"`
}

type configSetOutput struct {
	Path           string `json:"path"`
	Scope          string `json:"scope"`
	DefaultProfile string `json:"defaultProfile,omitempty"`
	Developer      string `json:"developer,omitempty"`
	PackageName    string `json:"packageName,omitempty"`
}

// GooglePlayConfigCommand returns config command group.
func GooglePlayConfigCommand() *ffcli.Command {
	fs := flag.NewFlagSet("config", flag.ExitOnError)

	return &ffcli.Command{
		Name:       "config",
		ShortUsage: "gplay config <subcommand> [flags]",
		ShortHelp:  "Manage persisted CLI defaults in ~/.gplay/config.json.",
		LongHelp: `Manage persisted CLI defaults in ~/.gplay/config.json.

Examples:
  gplay config set --developer 9092032418990165552 --package-name com.example.app
  gplay config set --profile prod --package-name com.example.app
  gplay config show --pretty`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			GooglePlayConfigSetCommand(),
			GooglePlayConfigShowCommand(),
		},
		Exec: func(ctx context.Context, args []string) error {
			return flag.ErrHelp
		},
	}
}

// GooglePlayConfigSetCommand persists defaults in config file.
func GooglePlayConfigSetCommand() *ffcli.Command {
	fs := flag.NewFlagSet("config set", flag.ExitOnError)

	targetProfile := fs.String("profile", "", "Profile to update for scoped defaults")
	defaultProfile := fs.String("default-profile", "", "Set default profile name")
	developer := fs.String("developer", "", "Set default developer ID/resource")
	packageName := fs.String("package-name", "", "Set default package name")
	output := shared.BindOutputFlags(fs)

	return &ffcli.Command{
		Name:       "set",
		ShortUsage: "gplay config set [--profile PROFILE] [--default-profile PROFILE] [--developer ID] [--package-name PACKAGE]",
		ShortHelp:  "Set persisted defaults for future commands.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			_ = ctx
			visited := map[string]bool{}
			fs.Visit(func(f *flag.Flag) {
				visited[f.Name] = true
			})

			updatesRequested := visited["default-profile"] || visited["developer"] || visited["package-name"]
			if !updatesRequested {
				fmt.Fprintln(os.Stderr, "Error: set at least one of --default-profile, --developer, or --package-name")
				return flag.ErrHelp
			}

			cfg, err := config.LoadOrDefault()
			if err != nil {
				return fmt.Errorf("googleplay config set: %w", err)
			}

			profileValue := strings.TrimSpace(*targetProfile)

			if visited["default-profile"] {
				value := strings.TrimSpace(*defaultProfile)
				if value == "" {
					fmt.Fprintln(os.Stderr, "Error: --default-profile cannot be empty")
					return flag.ErrHelp
				}
				cfg.DefaultProfile = value
			}

			if visited["developer"] {
				value := strings.TrimSpace(*developer)
				if value == "" {
					fmt.Fprintln(os.Stderr, "Error: --developer cannot be empty")
					return flag.ErrHelp
				}
				if profileValue != "" {
					profile := cfg.Profiles[profileValue]
					profile.Developer = value
					cfg.Profiles[profileValue] = profile
				} else {
					cfg.DefaultDeveloper = value
				}
			}

			if visited["package-name"] {
				value := strings.TrimSpace(*packageName)
				if value == "" {
					fmt.Fprintln(os.Stderr, "Error: --package-name cannot be empty")
					return flag.ErrHelp
				}
				if profileValue != "" {
					profile := cfg.Profiles[profileValue]
					profile.PackageName = value
					cfg.Profiles[profileValue] = profile
				} else {
					cfg.DefaultPackageName = value
				}
			}

			if err := config.Save(cfg); err != nil {
				return fmt.Errorf("googleplay config set: %w", err)
			}

			path, _ := config.Path()
			result := configSetOutput{
				Path:           path,
				Scope:          "global",
				DefaultProfile: cfg.DefaultProfile,
			}
			if profileValue != "" {
				result.Scope = "profile:" + profileValue
				profile := cfg.Profiles[profileValue]
				result.Developer = profile.Developer
				result.PackageName = profile.PackageName
			} else {
				result.Developer = cfg.DefaultDeveloper
				result.PackageName = cfg.DefaultPackageName
			}

			return shared.PrintOutput(result, *output.Output, *output.Pretty)
		},
	}
}

// GooglePlayConfigShowCommand prints config values and resolved defaults.
func GooglePlayConfigShowCommand() *ffcli.Command {
	fs := flag.NewFlagSet("config show", flag.ExitOnError)
	output := shared.BindOutputFlags(fs)

	return &ffcli.Command{
		Name:       "show",
		ShortUsage: "gplay config show [--pretty]",
		ShortHelp:  "Show current config file and resolved defaults.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			_ = ctx

			path, err := config.Path()
			if err != nil {
				return fmt.Errorf("googleplay config show: %w", err)
			}

			cfg, err := config.Load()
			exists := true
			if err != nil {
				if errors.Is(err, config.ErrNotFound) {
					exists = false
					cfg = &config.Config{Profiles: map[string]config.Profile{}}
				} else {
					return fmt.Errorf("googleplay config show: %w", err)
				}
			}

			resolvedDeveloper, _ := resolveDeveloper("")
			resolvedPackageName, _ := resolvePackageName("")

			result := configShowOutput{
				Path:                path,
				Exists:              exists,
				SelectedProfile:     shared.SelectedProfile(),
				ResolvedDeveloper:   strings.TrimSpace(resolvedDeveloper),
				ResolvedPackageName: strings.TrimSpace(resolvedPackageName),
				Config:              cfg,
			}

			return shared.PrintOutput(result, *output.Output, *output.Pretty)
		},
	}
}
