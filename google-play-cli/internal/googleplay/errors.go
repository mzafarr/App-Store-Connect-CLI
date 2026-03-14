package googleplay

import (
	"errors"
	"fmt"
	"strings"
)

var (
	ErrNotFound     = errors.New("resource not found")
	ErrUnauthorized = errors.New("unauthorized")
	ErrForbidden    = errors.New("forbidden")
	ErrBadRequest   = errors.New("bad request")
	ErrConflict     = errors.New("resource conflict")
)

// APIError represents a parsed Google Play API error.
type APIError struct {
	Code       string
	Title      string
	Detail     string
	StatusCode int
}

func (e *APIError) Error() string {
	title := strings.TrimSpace(e.Title)
	detail := strings.TrimSpace(e.Detail)
	code := strings.TrimSpace(e.Code)

	switch {
	case title != "" && detail != "":
		return fmt.Sprintf("%s: %s", title, detail)
	case title != "":
		return title
	case detail != "":
		return detail
	case code != "":
		return code
	default:
		return "API error"
	}
}

func (e *APIError) Is(target error) bool {
	switch target {
	case ErrNotFound:
		return strings.EqualFold(e.Code, "NOT_FOUND") || e.StatusCode == 404
	case ErrUnauthorized:
		return strings.EqualFold(e.Code, "UNAUTHORIZED") || e.StatusCode == 401
	case ErrForbidden:
		return strings.EqualFold(e.Code, "FORBIDDEN") || e.StatusCode == 403
	case ErrBadRequest:
		return strings.EqualFold(e.Code, "BAD_REQUEST") || e.StatusCode == 400
	case ErrConflict:
		return strings.EqualFold(e.Code, "CONFLICT") || e.StatusCode == 409
	default:
		return false
	}
}
