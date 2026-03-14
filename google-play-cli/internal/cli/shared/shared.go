package shared

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/peterbourgon/ff/v3/ffcli"
)

const (
	defaultTimeout         = 30 * time.Second
	defaultOutputEnvVar    = "GPCLI_DEFAULT_OUTPUT"
	timeoutEnvVar          = "GOOGLE_PLAY_TIMEOUT"
	timeoutSecondsEnvVar   = "GOOGLE_PLAY_TIMEOUT_SECONDS"
	profileEnvVar          = "GOOGLE_PLAY_PROFILE"
	supportedOutputFormats = "json, table"
)

var ErrMissingAuth = errors.New("missing authentication")

var selectedProfile string

// BindRootFlags registers root-level shared flags.
func BindRootFlags(fs *flag.FlagSet) {
	fs.StringVar(&selectedProfile, "profile", "", "Use named profile from config")
}

// SelectedProfile resolves profile from flag or env.
func SelectedProfile() string {
	if value := strings.TrimSpace(selectedProfile); value != "" {
		return value
	}
	return strings.TrimSpace(os.Getenv(profileEnvVar))
}

// OutputFlags stores pointers to output-related flag values.
type OutputFlags struct {
	Output *string
	Pretty *bool
}

// BindOutputFlags registers output flags.
func BindOutputFlags(fs *flag.FlagSet) OutputFlags {
	output := fs.String("output", DefaultOutputFormat(), "Output format: "+supportedOutputFormats)
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")
	return OutputFlags{Output: output, Pretty: pretty}
}

// DefaultOutputFormat resolves the default output format.
func DefaultOutputFormat() string {
	value := strings.ToLower(strings.TrimSpace(os.Getenv(defaultOutputEnvVar)))
	switch value {
	case "json", "table":
		return value
	default:
		return "json"
	}
}

// PrintOutput prints data in the selected output format.
func PrintOutput(data any, format string, pretty bool) error {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "", "json":
		return printJSON(data, pretty)
	case "table":
		// Table renderer is intentionally simple in this first slice.
		return printJSON(data, true)
	default:
		return fmt.Errorf("--output must be one of: %s", supportedOutputFormats)
	}
}

func printJSON(data any, pretty bool) error {
	enc := json.NewEncoder(os.Stdout)
	if pretty {
		enc.SetIndent("", "  ")
	}
	return enc.Encode(data)
}

// UsageError prints a usage-class error and returns flag.ErrHelp.
func UsageError(message string) error {
	trimmed := strings.TrimSpace(message)
	if trimmed != "" {
		fmt.Fprintf(os.Stderr, "Error: %s\n", trimmed)
	}
	return flag.ErrHelp
}

// ContextWithTimeout applies timeout controls from env vars.
func ContextWithTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	timeout := resolveTimeout()
	return context.WithTimeout(ctx, timeout)
}

func resolveTimeout() time.Duration {
	if raw := strings.TrimSpace(os.Getenv(timeoutEnvVar)); raw != "" {
		if parsed, err := time.ParseDuration(raw); err == nil && parsed > 0 {
			return parsed
		}
	}
	if raw := strings.TrimSpace(os.Getenv(timeoutSecondsEnvVar)); raw != "" {
		if seconds, err := strconv.Atoi(raw); err == nil && seconds > 0 {
			return time.Duration(seconds) * time.Second
		}
	}
	return defaultTimeout
}

// SplitCSV returns non-empty, trimmed CSV values.
func SplitCSV(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

// DefaultUsageFunc renders description, usage, subcommands, and flags.
func DefaultUsageFunc(c *ffcli.Command) string {
	var b strings.Builder

	if help := strings.TrimSpace(c.ShortHelp); help != "" {
		b.WriteString("DESCRIPTION\n")
		b.WriteString("  ")
		b.WriteString(help)
		b.WriteString("\n\n")
	}

	usage := strings.TrimSpace(c.ShortUsage)
	if usage == "" {
		usage = strings.TrimSpace(c.Name)
	}
	if usage != "" {
		b.WriteString("USAGE\n")
		b.WriteString("  ")
		b.WriteString(usage)
		b.WriteString("\n\n")
	}

	if len(c.Subcommands) > 0 {
		b.WriteString("SUBCOMMANDS\n")
		tw := tabwriter.NewWriter(&b, 0, 2, 2, ' ', 0)
		for _, sub := range c.Subcommands {
			_, _ = fmt.Fprintf(tw, "  %s\t%s\n", sub.Name, sub.ShortHelp)
		}
		_ = tw.Flush()
		b.WriteString("\n")
	}

	if c.FlagSet != nil {
		hasFlags := false
		c.FlagSet.VisitAll(func(*flag.Flag) { hasFlags = true })
		if hasFlags {
			b.WriteString("FLAGS\n")
			tw := tabwriter.NewWriter(&b, 0, 2, 2, ' ', 0)
			c.FlagSet.VisitAll(func(f *flag.Flag) {
				if f.DefValue != "" {
					_, _ = fmt.Fprintf(tw, "  --%s\t%s (default: %s)\n", f.Name, f.Usage, f.DefValue)
					return
				}
				_, _ = fmt.Fprintf(tw, "  --%s\t%s\n", f.Name, f.Usage)
			})
			_ = tw.Flush()
			b.WriteString("\n")
		}
	}

	return b.String()
}
