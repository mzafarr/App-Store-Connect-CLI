package cmd

import (
	"errors"
	"flag"
	"net/http"

	"github.com/rudrankriyam/google-play-cli/internal/cli/shared"
	"github.com/rudrankriyam/google-play-cli/internal/googleplay"
)

const (
	ExitSuccess  = 0
	ExitError    = 1
	ExitUsage    = 2
	ExitAuth     = 3
	ExitNotFound = 4
	ExitConflict = 5
)

func ExitCodeFromError(err error) int {
	if err == nil {
		return ExitSuccess
	}
	if errors.Is(err, flag.ErrHelp) {
		return ExitUsage
	}
	if errors.Is(err, shared.ErrMissingAuth) ||
		errors.Is(err, googleplay.ErrUnauthorized) ||
		errors.Is(err, googleplay.ErrForbidden) {
		return ExitAuth
	}
	if errors.Is(err, googleplay.ErrNotFound) {
		return ExitNotFound
	}
	if errors.Is(err, googleplay.ErrConflict) {
		return ExitConflict
	}

	var apiErr *googleplay.APIError
	if errors.As(err, &apiErr) && apiErr.StatusCode > 0 {
		return HTTPStatusToExitCode(apiErr.StatusCode)
	}

	return ExitError
}

func HTTPStatusToExitCode(status int) int {
	switch status {
	case http.StatusUnauthorized, http.StatusForbidden:
		return ExitAuth
	case http.StatusNotFound:
		return ExitNotFound
	case http.StatusConflict:
		return ExitConflict
	case http.StatusBadRequest:
		return 10
	default:
		if status >= 500 && status <= 599 {
			return min(60+(status-500), 99)
		}
		return ExitError
	}
}
