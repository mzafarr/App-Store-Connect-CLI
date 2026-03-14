package googleplay

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/rudrankriyam/google-play-cli/internal/config"
)

const (
	// ServiceAccountPathEnvVar points to a service-account JSON key file.
	ServiceAccountPathEnvVar = "GOOGLE_PLAY_SERVICE_ACCOUNT_PATH"
	// ServiceAccountJSONEnvVar stores service-account JSON content directly.
	ServiceAccountJSONEnvVar = "GOOGLE_PLAY_SERVICE_ACCOUNT_JSON"
	// TokenURLOverrideEnvVar overrides the OAuth token URL.
	TokenURLOverrideEnvVar = "GOOGLE_PLAY_TOKEN_URL"
	// APIBaseURLOverrideEnvVar overrides the Android Publisher API base URL.
	APIBaseURLOverrideEnvVar = "GOOGLE_PLAY_API_BASE_URL"

	defaultTokenURL   = "https://oauth2.googleapis.com/token"
	defaultAPIBaseURL = "https://androidpublisher.googleapis.com"

	androidPublisherScope = "https://www.googleapis.com/auth/androidpublisher"
)

// ErrMissingCredentials indicates that no Google Play credentials were found.
var ErrMissingCredentials = errors.New("missing google play service account credentials")

// CredentialSource describes where credentials were resolved from.
type CredentialSource struct {
	Kind    string `json:"kind"`
	Profile string `json:"profile,omitempty"`
	Path    string `json:"path,omitempty"`
}

// LoadServiceAccount resolves credentials from env first, then optional profile config.
func LoadServiceAccount(profile string) (ServiceAccount, CredentialSource, error) {
	if path := strings.TrimSpace(os.Getenv(ServiceAccountPathEnvVar)); path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return ServiceAccount{}, CredentialSource{}, fmt.Errorf("googleplay auth: read %s: %w", ServiceAccountPathEnvVar, err)
		}
		account, err := parseServiceAccountJSON(data)
		if err != nil {
			return ServiceAccount{}, CredentialSource{}, err
		}
		return account, CredentialSource{Kind: "env_path", Path: path}, nil
	}

	if rawJSON := strings.TrimSpace(os.Getenv(ServiceAccountJSONEnvVar)); rawJSON != "" {
		account, err := parseServiceAccountJSON([]byte(rawJSON))
		if err != nil {
			return ServiceAccount{}, CredentialSource{}, err
		}
		return account, CredentialSource{Kind: "env_json"}, nil
	}

	if cfg, err := config.Load(); err == nil {
		resolvedName, resolvedProfile, err := cfg.ResolveProfile(profile)
		if err != nil {
			return ServiceAccount{}, CredentialSource{}, fmt.Errorf("googleplay auth: %w", err)
		}

		if path := strings.TrimSpace(resolvedProfile.ServiceAccountPath); path != "" {
			data, readErr := os.ReadFile(path)
			if readErr != nil {
				return ServiceAccount{}, CredentialSource{}, fmt.Errorf("googleplay auth: read profile %q service account path: %w", resolvedName, readErr)
			}
			account, parseErr := parseServiceAccountJSON(data)
			if parseErr != nil {
				return ServiceAccount{}, CredentialSource{}, parseErr
			}
			applyProfileOverrides(&account, resolvedProfile)
			return account, CredentialSource{Kind: "profile_path", Profile: resolvedName, Path: path}, nil
		}

		if raw := strings.TrimSpace(resolvedProfile.ServiceAccountJSON); raw != "" {
			account, parseErr := parseServiceAccountJSON([]byte(raw))
			if parseErr != nil {
				return ServiceAccount{}, CredentialSource{}, parseErr
			}
			applyProfileOverrides(&account, resolvedProfile)
			return account, CredentialSource{Kind: "profile_json", Profile: resolvedName}, nil
		}

		return ServiceAccount{}, CredentialSource{}, fmt.Errorf("googleplay auth: profile %q is missing service_account_path/service_account_json", resolvedName)
	} else if !errors.Is(err, config.ErrNotFound) {
		return ServiceAccount{}, CredentialSource{}, fmt.Errorf("googleplay auth: load config: %w", err)
	}

	return ServiceAccount{}, CredentialSource{}, fmt.Errorf(
		"%w: set %s or %s",
		ErrMissingCredentials,
		ServiceAccountPathEnvVar,
		ServiceAccountJSONEnvVar,
	)
}

// LoadServiceAccountFromEnv resolves credentials from env variables only.
func LoadServiceAccountFromEnv() (ServiceAccount, error) {
	if path := strings.TrimSpace(os.Getenv(ServiceAccountPathEnvVar)); path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return ServiceAccount{}, fmt.Errorf("googleplay auth: read %s: %w", ServiceAccountPathEnvVar, err)
		}
		return parseServiceAccountJSON(data)
	}
	if rawJSON := strings.TrimSpace(os.Getenv(ServiceAccountJSONEnvVar)); rawJSON != "" {
		return parseServiceAccountJSON([]byte(rawJSON))
	}
	return ServiceAccount{}, fmt.Errorf(
		"%w: set %s or %s",
		ErrMissingCredentials,
		ServiceAccountPathEnvVar,
		ServiceAccountJSONEnvVar,
	)
}

func parseServiceAccountJSON(data []byte) (ServiceAccount, error) {
	var account ServiceAccount
	if err := json.Unmarshal(data, &account); err != nil {
		return ServiceAccount{}, fmt.Errorf("googleplay auth: parse service account JSON: %w", err)
	}

	account.ClientEmail = strings.TrimSpace(account.ClientEmail)
	account.PrivateKey = normalizeMultilineSecret(account.PrivateKey)
	account.TokenURI = strings.TrimSpace(account.TokenURI)

	if account.ClientEmail == "" {
		return ServiceAccount{}, fmt.Errorf("googleplay auth: service account is missing client_email")
	}
	if strings.TrimSpace(account.PrivateKey) == "" {
		return ServiceAccount{}, fmt.Errorf("googleplay auth: service account is missing private_key")
	}
	if account.TokenURI == "" {
		account.TokenURI = defaultTokenURL
	}

	return account, nil
}

func applyProfileOverrides(account *ServiceAccount, profile config.Profile) {
	if account == nil {
		return
	}
	if tokenURL := strings.TrimSpace(profile.TokenURL); tokenURL != "" {
		account.TokenURI = tokenURL
	}
	if apiBaseURL := strings.TrimSpace(profile.APIBaseURL); apiBaseURL != "" {
		account.APIBaseURL = strings.TrimRight(apiBaseURL, "/")
	}
}

func normalizeMultilineSecret(value string) string {
	if strings.Contains(value, "\\n") && !strings.Contains(value, "\n") {
		return strings.ReplaceAll(value, "\\n", "\n")
	}
	return value
}

func resolveTokenURL(accountTokenURL string) string {
	if override := strings.TrimSpace(os.Getenv(TokenURLOverrideEnvVar)); override != "" {
		return override
	}
	if v := strings.TrimSpace(accountTokenURL); v != "" {
		return v
	}
	return defaultTokenURL
}

func resolveAPIBaseURL(profileAPIBaseURL string) string {
	if override := strings.TrimSpace(os.Getenv(APIBaseURLOverrideEnvVar)); override != "" {
		return strings.TrimRight(override, "/")
	}
	if profileAPIBaseURL = strings.TrimSpace(profileAPIBaseURL); profileAPIBaseURL != "" {
		return strings.TrimRight(profileAPIBaseURL, "/")
	}
	return defaultAPIBaseURL
}
