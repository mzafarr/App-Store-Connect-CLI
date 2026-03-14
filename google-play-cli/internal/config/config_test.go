package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadReturnsNotFoundWhenMissing(t *testing.T) {
	t.Setenv(configPathEnvVar, filepath.Join(t.TempDir(), "missing.json"))

	_, err := Load()
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestResolveProfileUsesDefault(t *testing.T) {
	cfg := &Config{
		DefaultProfile: "prod",
		Profiles: map[string]Profile{
			"prod": {ServiceAccountPath: "/tmp/prod.json"},
			"dev":  {ServiceAccountPath: "/tmp/dev.json"},
		},
	}

	name, profile, err := cfg.ResolveProfile("")
	if err != nil {
		t.Fatalf("ResolveProfile() error: %v", err)
	}
	if name != "prod" {
		t.Fatalf("profile name = %q, want prod", name)
	}
	if profile.ServiceAccountPath != "/tmp/prod.json" {
		t.Fatalf("profile path = %q", profile.ServiceAccountPath)
	}
}

func TestLoadAndResolveProfile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	raw := `{
  "default_profile": "ci",
  "profiles": {
    "ci": {
      "service_account_path": "/tmp/sa.json",
      "token_url": "https://token.example.test"
    }
  }
}`
	if err := os.WriteFile(path, []byte(raw), 0o600); err != nil {
		t.Fatalf("WriteFile() error: %v", err)
	}
	t.Setenv(configPathEnvVar, path)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	name, profile, err := cfg.ResolveProfile("")
	if err != nil {
		t.Fatalf("ResolveProfile() error: %v", err)
	}
	if name != "ci" {
		t.Fatalf("name = %q, want ci", name)
	}
	if profile.TokenURL != "https://token.example.test" {
		t.Fatalf("token_url = %q", profile.TokenURL)
	}
}

func TestLoadOrDefaultReturnsEmptyConfigWhenMissing(t *testing.T) {
	t.Setenv(configPathEnvVar, filepath.Join(t.TempDir(), "missing.json"))

	cfg, err := LoadOrDefault()
	if err != nil {
		t.Fatalf("LoadOrDefault() error: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected non-nil config")
	}
	if cfg.Profiles == nil {
		t.Fatal("expected profiles map initialized")
	}
}

func TestSaveAndReloadConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	t.Setenv(configPathEnvVar, path)

	cfg := &Config{
		DefaultProfile:     "prod",
		DefaultDeveloper:   "9092032418990165552",
		DefaultPackageName: "com.example.app",
		Profiles: map[string]Profile{
			"prod": {
				ServiceAccountPath: "/tmp/prod.json",
				Developer:          "developers/9092032418990165552",
				PackageName:        "com.example.app",
			},
		},
	}
	if err := Save(cfg); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	reloaded, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if reloaded.DefaultDeveloper != "9092032418990165552" {
		t.Fatalf("default developer = %q", reloaded.DefaultDeveloper)
	}
	if reloaded.DefaultPackageName != "com.example.app" {
		t.Fatalf("default package = %q", reloaded.DefaultPackageName)
	}
	if reloaded.Profiles["prod"].PackageName != "com.example.app" {
		t.Fatalf("profile package = %q", reloaded.Profiles["prod"].PackageName)
	}
}
