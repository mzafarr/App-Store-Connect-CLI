package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	configPathEnvVar = "GPCLI_CONFIG_PATH"
	configDirName    = ".gplay"
	configFileName   = "config.json"
)

var (
	// ErrNotFound indicates that config file does not exist.
	ErrNotFound = errors.New("configuration not found")
	// ErrInvalid indicates malformed config.
	ErrInvalid = errors.New("invalid configuration")
)

// Profile stores per-profile credential and endpoint settings.
type Profile struct {
	ServiceAccountPath string `json:"service_account_path,omitempty"`
	ServiceAccountJSON string `json:"service_account_json,omitempty"`
	TokenURL           string `json:"token_url,omitempty"`
	APIBaseURL         string `json:"api_base_url,omitempty"`
	Developer          string `json:"developer,omitempty"`
	PackageName        string `json:"package_name,omitempty"`
}

// Config is the root config shape.
type Config struct {
	DefaultProfile     string             `json:"default_profile,omitempty"`
	DefaultDeveloper   string             `json:"default_developer,omitempty"`
	DefaultPackageName string             `json:"default_package_name,omitempty"`
	Profiles           map[string]Profile `json:"profiles,omitempty"`
}

// Path returns active config path.
func Path() (string, error) {
	if custom := strings.TrimSpace(os.Getenv(configPathEnvVar)); custom != "" {
		resolved, err := filepath.Abs(custom)
		if err != nil {
			return "", fmt.Errorf("%w: %s must be a valid path", ErrInvalid, configPathEnvVar)
		}
		return resolved, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	return filepath.Join(home, configDirName, configFileName), nil
}

// Load reads config from disk.
func Load() (*Config, error) {
	path, err := Path()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("%w: parse JSON: %v", ErrInvalid, err)
	}
	ensureProfilesMap(&cfg)
	return &cfg, nil
}

// LoadOrDefault reads config from disk or returns an empty config when missing.
func LoadOrDefault() (*Config, error) {
	cfg, err := Load()
	if err == nil {
		return cfg, nil
	}
	if errors.Is(err, ErrNotFound) {
		return &Config{Profiles: map[string]Profile{}}, nil
	}
	return nil, err
}

// Save persists config to disk.
func Save(cfg *Config) error {
	if cfg == nil {
		return fmt.Errorf("%w: config is nil", ErrInvalid)
	}
	path, err := Path()
	if err != nil {
		return err
	}
	ensureProfilesMap(cfg)

	raw, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	raw = append(raw, '\n')

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

// ResolveProfile returns selected profile by explicit name/default/single-profile fallback.
func (c *Config) ResolveProfile(name string) (string, Profile, error) {
	if c == nil || len(c.Profiles) == 0 {
		return "", Profile{}, fmt.Errorf("%w: no profiles configured", ErrNotFound)
	}

	target := strings.TrimSpace(name)
	if target == "" {
		target = strings.TrimSpace(c.DefaultProfile)
	}
	if target == "" {
		if len(c.Profiles) == 1 {
			for key, profile := range c.Profiles {
				return key, profile, nil
			}
		}
		return "", Profile{}, fmt.Errorf("%w: multiple profiles configured; pass --profile or set GOOGLE_PLAY_PROFILE", ErrInvalid)
	}

	profile, ok := c.Profiles[target]
	if !ok {
		return "", Profile{}, fmt.Errorf("%w: profile %q not found", ErrInvalid, target)
	}
	return target, profile, nil
}

func ensureProfilesMap(cfg *Config) {
	if cfg != nil && cfg.Profiles == nil {
		cfg.Profiles = map[string]Profile{}
	}
}
