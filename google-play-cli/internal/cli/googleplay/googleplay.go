package googleplay

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/rudrankriyam/google-play-cli/internal/cli/shared"
	"github.com/rudrankriyam/google-play-cli/internal/config"
	play "github.com/rudrankriyam/google-play-cli/internal/googleplay"
)

const (
	developerEnvVar   = "GOOGLE_PLAY_DEVELOPER"
	packageNameEnvVar = "GOOGLE_PLAY_PACKAGE_NAME"
)

func newGooglePlayClient() (*play.Client, error) {
	client, _, err := play.NewClientFromProfile(shared.SelectedProfile())
	if err == nil {
		return client, nil
	}

	if errors.Is(err, play.ErrMissingCredentials) {
		return nil, fmt.Errorf("%w: %v", shared.ErrMissingAuth, err)
	}

	return nil, err
}

func loadGooglePlayCredentials() (play.ServiceAccount, play.CredentialSource, error) {
	account, source, err := play.LoadServiceAccount(shared.SelectedProfile())
	if err != nil {
		if errors.Is(err, play.ErrMissingCredentials) {
			return play.ServiceAccount{}, play.CredentialSource{}, fmt.Errorf("%w: %v", shared.ErrMissingAuth, err)
		}
		return play.ServiceAccount{}, play.CredentialSource{}, err
	}
	return account, source, nil
}

func resolvePackageName(value string) (string, error) {
	if resolved := strings.TrimSpace(value); resolved != "" {
		return resolved, nil
	}
	if resolved := strings.TrimSpace(os.Getenv(packageNameEnvVar)); resolved != "" {
		return resolved, nil
	}

	cfg, err := loadOptionalConfig()
	if err != nil {
		return "", err
	}
	if cfg == nil {
		return "", nil
	}

	fromProfile, err := resolveProfileValue(cfg, func(p config.Profile) string { return p.PackageName })
	if err != nil {
		return "", err
	}
	if fromProfile != "" {
		return fromProfile, nil
	}

	return strings.TrimSpace(cfg.DefaultPackageName), nil
}

func resolveDeveloper(value string) (string, error) {
	if resolved := strings.TrimSpace(value); resolved != "" {
		return resolved, nil
	}
	if resolved := strings.TrimSpace(os.Getenv(developerEnvVar)); resolved != "" {
		return resolved, nil
	}

	cfg, err := loadOptionalConfig()
	if err != nil {
		return "", err
	}
	if cfg == nil {
		return "", nil
	}

	fromProfile, err := resolveProfileValue(cfg, func(p config.Profile) string { return p.Developer })
	if err != nil {
		return "", err
	}
	if fromProfile != "" {
		return fromProfile, nil
	}

	return strings.TrimSpace(cfg.DefaultDeveloper), nil
}

func loadOptionalConfig() (*config.Config, error) {
	cfg, err := config.Load()
	if err == nil {
		return cfg, nil
	}
	if errors.Is(err, config.ErrNotFound) {
		return nil, nil
	}
	return nil, fmt.Errorf("googleplay config: %w", err)
}

func resolveProfileValue(cfg *config.Config, getter func(config.Profile) string) (string, error) {
	if cfg == nil {
		return "", nil
	}

	selected := strings.TrimSpace(shared.SelectedProfile())
	if selected != "" {
		_, profile, err := cfg.ResolveProfile(selected)
		if err != nil {
			return "", fmt.Errorf("googleplay config: %w", err)
		}
		return strings.TrimSpace(getter(profile)), nil
	}

	_, profile, err := cfg.ResolveProfile("")
	if err == nil {
		return strings.TrimSpace(getter(profile)), nil
	}
	if errors.Is(err, config.ErrInvalid) || errors.Is(err, config.ErrNotFound) {
		return "", nil
	}
	return "", fmt.Errorf("googleplay config: %w", err)
}
