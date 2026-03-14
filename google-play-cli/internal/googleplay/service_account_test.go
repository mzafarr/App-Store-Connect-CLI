package googleplay

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadServiceAccountFromEnv_Missing(t *testing.T) {
	t.Setenv(ServiceAccountPathEnvVar, "")
	t.Setenv(ServiceAccountJSONEnvVar, "")

	_, err := LoadServiceAccountFromEnv()
	if !errors.Is(err, ErrMissingCredentials) {
		t.Fatalf("expected ErrMissingCredentials, got %v", err)
	}
}

func TestLoadServiceAccountFromEnv_PathPreferredOverInlineJSON(t *testing.T) {
	t.Setenv(ServiceAccountJSONEnvVar, `{"client_email":"inline@example.com","private_key":"inline"}`)

	file := filepath.Join(t.TempDir(), "service-account.json")
	if err := os.WriteFile(file, []byte(sampleServiceAccountJSON("file@example.com", "-----BEGIN PRIVATE KEY-----\nA\n-----END PRIVATE KEY-----\n")), 0o600); err != nil {
		t.Fatalf("WriteFile() error: %v", err)
	}
	t.Setenv(ServiceAccountPathEnvVar, file)

	account, err := LoadServiceAccountFromEnv()
	if err != nil {
		t.Fatalf("LoadServiceAccountFromEnv() error: %v", err)
	}
	if account.ClientEmail != "file@example.com" {
		t.Fatalf("client email = %q, want %q", account.ClientEmail, "file@example.com")
	}
}

func TestLoadServiceAccountFromEnv_NormalizesEscapedNewlines(t *testing.T) {
	t.Setenv(ServiceAccountPathEnvVar, "")
	t.Setenv(ServiceAccountJSONEnvVar, sampleServiceAccountJSON("example@test.local", "-----BEGIN PRIVATE KEY-----\\nABC\\n-----END PRIVATE KEY-----\\n"))

	account, err := LoadServiceAccountFromEnv()
	if err != nil {
		t.Fatalf("LoadServiceAccountFromEnv() error: %v", err)
	}
	if !strings.Contains(account.PrivateKey, "\nABC\n") {
		t.Fatalf("expected normalized newlines in private key, got %q", account.PrivateKey)
	}
	if account.TokenURI != defaultTokenURL {
		t.Fatalf("token URI = %q, want %q", account.TokenURI, defaultTokenURL)
	}
}

func TestLoadServiceAccount_UsesProfileConfigPath(t *testing.T) {
	t.Setenv(ServiceAccountPathEnvVar, "")
	t.Setenv(ServiceAccountJSONEnvVar, "")

	tmpDir := t.TempDir()
	serviceAccountPath := filepath.Join(tmpDir, "play-prod.json")
	if err := os.WriteFile(
		serviceAccountPath,
		[]byte(sampleServiceAccountJSON("profile@example.com", "-----BEGIN PRIVATE KEY-----\nA\n-----END PRIVATE KEY-----\n")),
		0o600,
	); err != nil {
		t.Fatalf("WriteFile(service-account) error: %v", err)
	}

	configPath := filepath.Join(tmpDir, "config.json")
	configJSON := `{
  "default_profile": "prod",
  "profiles": {
    "prod": {
      "service_account_path": "` + strings.ReplaceAll(serviceAccountPath, `\`, `\\`) + `",
      "token_url": "https://token.example.test",
      "api_base_url": "https://androidpublisher.example.test/"
    }
  }
}`
	if err := os.WriteFile(configPath, []byte(configJSON), 0o600); err != nil {
		t.Fatalf("WriteFile(config) error: %v", err)
	}
	t.Setenv("GPCLI_CONFIG_PATH", configPath)

	account, source, err := LoadServiceAccount("")
	if err != nil {
		t.Fatalf("LoadServiceAccount() error: %v", err)
	}
	if source.Kind != "profile_path" {
		t.Fatalf("source kind = %q, want profile_path", source.Kind)
	}
	if source.Profile != "prod" {
		t.Fatalf("source profile = %q, want prod", source.Profile)
	}
	if source.Path != serviceAccountPath {
		t.Fatalf("source path = %q, want %q", source.Path, serviceAccountPath)
	}
	if account.ClientEmail != "profile@example.com" {
		t.Fatalf("client email = %q", account.ClientEmail)
	}
	if account.TokenURI != "https://token.example.test" {
		t.Fatalf("token uri = %q", account.TokenURI)
	}
	if account.APIBaseURL != "https://androidpublisher.example.test" {
		t.Fatalf("api base url = %q", account.APIBaseURL)
	}
}

func TestLoadServiceAccount_MissingProfileSelectionReturnsError(t *testing.T) {
	t.Setenv(ServiceAccountPathEnvVar, "")
	t.Setenv(ServiceAccountJSONEnvVar, "")

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	configJSON := `{
  "profiles": {
    "prod": {"service_account_json": "{}"},
    "dev": {"service_account_json": "{}"}
  }
}`
	if err := os.WriteFile(configPath, []byte(configJSON), 0o600); err != nil {
		t.Fatalf("WriteFile(config) error: %v", err)
	}
	t.Setenv("GPCLI_CONFIG_PATH", configPath)

	_, _, err := LoadServiceAccount("")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "multiple profiles configured") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func sampleServiceAccountJSON(email, privateKey string) string {
	payload := map[string]any{
		"type":                        "service_account",
		"project_id":                  "demo-project",
		"private_key_id":              "key-id",
		"private_key":                 privateKey,
		"client_email":                email,
		"client_id":                   "1234567890",
		"auth_uri":                    "https://accounts.google.com/o/oauth2/auth",
		"token_uri":                   "",
		"auth_provider_x509_cert_url": "https://www.googleapis.com/oauth2/v1/certs",
		"client_x509_cert_url":        "https://www.googleapis.com/robot/v1/metadata/x509/demo",
	}
	data, _ := json.Marshal(payload)
	return string(data)
}
