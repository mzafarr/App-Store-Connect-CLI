package googleplay

import (
	"context"
	"flag"
	"fmt"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/rudrankriyam/google-play-cli/internal/cli/shared"
	play "github.com/rudrankriyam/google-play-cli/internal/googleplay"
)

type authDoctorOutput struct {
	OK        bool   `json:"ok"`
	Profile   string `json:"profile,omitempty"`
	Source    string `json:"source,omitempty"`
	Path      string `json:"path,omitempty"`
	Email     string `json:"email,omitempty"`
	ProjectID string `json:"projectId,omitempty"`
	Message   string `json:"message,omitempty"`
}

// GooglePlayAuthCommand returns auth command group.
func GooglePlayAuthCommand() *ffcli.Command {
	fs := flag.NewFlagSet("auth", flag.ExitOnError)

	return &ffcli.Command{
		Name:       "auth",
		ShortUsage: "gplay auth <subcommand> [flags]",
		ShortHelp:  "Inspect Google Play authentication configuration.",
		LongHelp: `Inspect Google Play authentication configuration.

Examples:
  gplay auth doctor
  gplay --profile prod auth doctor --output json --pretty`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			GooglePlayAuthDoctorCommand(),
		},
		Exec: func(ctx context.Context, args []string) error {
			return flag.ErrHelp
		},
	}
}

// GooglePlayAuthDoctorCommand validates auth resolution and key material.
func GooglePlayAuthDoctorCommand() *ffcli.Command {
	fs := flag.NewFlagSet("auth doctor", flag.ExitOnError)
	output := shared.BindOutputFlags(fs)

	return &ffcli.Command{
		Name:       "doctor",
		ShortUsage: "gplay auth doctor [--output json --pretty]",
		ShortHelp:  "Validate resolved credentials and print source details.",
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			_ = ctx

			profile := shared.SelectedProfile()
			account, source, err := loadGooglePlayCredentials()
			if err != nil {
				result := authDoctorOutput{
					OK:      false,
					Profile: profile,
					Message: err.Error(),
				}
				_ = shared.PrintOutput(result, *output.Output, *output.Pretty)
				return err
			}

			if err := play.ValidateServiceAccount(account); err != nil {
				result := authDoctorOutput{
					OK:      false,
					Profile: profile,
					Source:  source.Kind,
					Path:    source.Path,
					Email:   account.ClientEmail,
					Message: err.Error(),
				}
				_ = shared.PrintOutput(result, *output.Output, *output.Pretty)
				return fmt.Errorf("auth doctor: %w", err)
			}

			result := authDoctorOutput{
				OK:        true,
				Profile:   profile,
				Source:    source.Kind,
				Path:      source.Path,
				Email:     account.ClientEmail,
				ProjectID: account.ProjectID,
				Message:   "credentials resolved and validated",
			}
			if source.Profile != "" {
				result.Profile = source.Profile
			}

			if err := shared.PrintOutput(result, *output.Output, *output.Pretty); err != nil {
				return fmt.Errorf("auth doctor: %w", err)
			}
			return nil
		},
	}
}
