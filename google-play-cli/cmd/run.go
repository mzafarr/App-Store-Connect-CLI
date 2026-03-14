package cmd

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
)

// Run executes the CLI and returns the intended process exit code.
func Run(args []string, version string) int {
	root := RootCommand(version)

	if err := root.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return ExitSuccess
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return ExitCodeFromError(err)
	}

	if len(args) == 0 {
		fmt.Fprint(os.Stdout, root.UsageFunc(root))
		return ExitSuccess
	}

	if err := root.Run(context.Background()); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return ExitUsage
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return ExitCodeFromError(err)
	}

	return ExitSuccess
}
