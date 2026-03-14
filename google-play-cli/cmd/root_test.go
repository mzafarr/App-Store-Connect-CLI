package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func captureOutput(t *testing.T, fn func()) (string, string) {
	t.Helper()

	oldStdout := os.Stdout
	oldStderr := os.Stderr

	rOut, wOut, err := os.Pipe()
	if err != nil {
		t.Fatalf("stdout pipe error: %v", err)
	}
	rErr, wErr, err := os.Pipe()
	if err != nil {
		t.Fatalf("stderr pipe error: %v", err)
	}

	os.Stdout = wOut
	os.Stderr = wErr

	outC := make(chan string)
	errC := make(chan string)

	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, rOut)
		_ = rOut.Close()
		outC <- buf.String()
	}()
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, rErr)
		_ = rErr.Close()
		errC <- buf.String()
	}()

	fn()

	_ = wOut.Close()
	_ = wErr.Close()
	stdout := <-outC
	stderr := <-errC

	os.Stdout = oldStdout
	os.Stderr = oldStderr

	return stdout, stderr
}

func withTempConfigPath(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.json")
	t.Setenv("GPCLI_CONFIG_PATH", path)
	return path
}

func TestEditsCreateRequiresPackageName(t *testing.T) {
	withTempConfigPath(t)
	t.Setenv("GOOGLE_PLAY_PACKAGE_NAME", "")

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{"edits", "create"}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		err := root.Run(context.Background())
		if !errors.Is(err, flag.ErrHelp) {
			t.Fatalf("expected ErrHelp, got %v", err)
		}
	})

	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "Error: --package-name is required") {
		t.Fatalf("expected package-name error, got %q", stderr)
	}
}

func TestTracksUpdateRequiresVersionCodes(t *testing.T) {
	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{
			"tracks", "update",
			"--package-name", "com.example.app",
			"--edit-id", "edit-1",
			"--track", "production",
			"--status", "completed",
		}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		err := root.Run(context.Background())
		if !errors.Is(err, flag.ErrHelp) {
			t.Fatalf("expected ErrHelp, got %v", err)
		}
	})

	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "Error: --version-codes is required") {
		t.Fatalf("expected version-codes error, got %q", stderr)
	}
}

func TestTracksUpdateRequiresConfirmProduction(t *testing.T) {
	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{
			"tracks", "update",
			"--package-name", "com.example.app",
			"--edit-id", "edit-1",
			"--track", "production",
			"--version-codes", "123",
			"--status", "completed",
		}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		err := root.Run(context.Background())
		if !errors.Is(err, flag.ErrHelp) {
			t.Fatalf("expected ErrHelp, got %v", err)
		}
	})

	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "Error: --confirm-production is required when --track is production") {
		t.Fatalf("expected confirm-production validation error, got %q", stderr)
	}
}

func TestBundlesUploadRequiresAAB(t *testing.T) {
	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{
			"bundles", "upload",
			"--package-name", "com.example.app",
			"--edit-id", "edit-1",
		}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		err := root.Run(context.Background())
		if !errors.Is(err, flag.ErrHelp) {
			t.Fatalf("expected ErrHelp, got %v", err)
		}
	})

	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "Error: --aab is required") {
		t.Fatalf("expected --aab validation error, got %q", stderr)
	}
}

func TestReleaseRunRequiresAABOrVersionCodes(t *testing.T) {
	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{
			"release", "run",
			"--package-name", "com.example.app",
			"--track", "production",
			"--status", "completed",
		}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		err := root.Run(context.Background())
		if !errors.Is(err, flag.ErrHelp) {
			t.Fatalf("expected ErrHelp, got %v", err)
		}
	})

	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "Error: exactly one of --aab, --apk, or --version-codes is required") {
		t.Fatalf("expected release input validation error, got %q", stderr)
	}
}

func TestReleaseRunRejectsMutuallyExclusiveInputs(t *testing.T) {
	tmpDir := t.TempDir()
	aabPath := filepath.Join(tmpDir, "app.aab")
	if err := os.WriteFile(aabPath, []byte("aab"), 0o600); err != nil {
		t.Fatalf("WriteFile() error: %v", err)
	}

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{
			"release", "run",
			"--package-name", "com.example.app",
			"--track", "production",
			"--status", "completed",
			"--aab", aabPath,
			"--version-codes", "123",
		}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		err := root.Run(context.Background())
		if !errors.Is(err, flag.ErrHelp) {
			t.Fatalf("expected ErrHelp, got %v", err)
		}
	})

	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "Error: --aab, --apk, and --version-codes are mutually exclusive") {
		t.Fatalf("expected mutually exclusive error, got %q", stderr)
	}
}

func TestReleaseRunRequiresConfirmProduction(t *testing.T) {
	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{
			"release", "run",
			"--package-name", "com.example.app",
			"--track", "production",
			"--status", "completed",
			"--version-codes", "123",
		}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		err := root.Run(context.Background())
		if !errors.Is(err, flag.ErrHelp) {
			t.Fatalf("expected ErrHelp, got %v", err)
		}
	})

	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "Error: --confirm-production is required when --track is production") {
		t.Fatalf("expected confirm-production validation error, got %q", stderr)
	}
}

func TestReleaseRunDryRunWithAPKInput(t *testing.T) {
	tmpDir := t.TempDir()
	apkPath := filepath.Join(tmpDir, "app-release.apk")
	if err := os.WriteFile(apkPath, []byte("apk"), 0o600); err != nil {
		t.Fatalf("WriteFile() error: %v", err)
	}

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{
			"release", "run",
			"--package-name", "com.example.app",
			"--track", "production",
			"--status", "completed",
			"--apk", apkPath,
			"--dry-run",
		}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		if err := root.Run(context.Background()); err != nil {
			t.Fatalf("unexpected run error: %v", err)
		}
	})

	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(stdout, `"dryRun": true`) && !strings.Contains(stdout, `"dryRun":true`) {
		t.Fatalf("expected dryRun output, got %q", stdout)
	}
}

func TestReleaseRunRejectsInvalidUntilStatus(t *testing.T) {
	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{
			"release", "run",
			"--package-name", "com.example.app",
			"--track", "production",
			"--status", "completed",
			"--version-codes", "123",
			"--wait",
			"--until-status", "unknown",
			"--dry-run",
		}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		err := root.Run(context.Background())
		if !errors.Is(err, flag.ErrHelp) {
			t.Fatalf("expected ErrHelp, got %v", err)
		}
	})

	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "Error: --until-status must be one of: draft, inProgress, halted, completed") {
		t.Fatalf("expected until-status validation error, got %q", stderr)
	}
}

func TestReleaseRunDryRunSkipsAuth(t *testing.T) {
	t.Setenv("GOOGLE_PLAY_SERVICE_ACCOUNT_PATH", "")
	t.Setenv("GOOGLE_PLAY_SERVICE_ACCOUNT_JSON", "")

	root := RootCommand("1.2.3")

	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{
			"release", "run",
			"--package-name", "com.example.app",
			"--track", "production",
			"--status", "completed",
			"--version-codes", "123",
			"--dry-run",
		}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		if err := root.Run(context.Background()); err != nil {
			t.Fatalf("unexpected run error: %v", err)
		}
	})

	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(stdout, `"dryRun": true`) && !strings.Contains(stdout, `"dryRun":true`) {
		t.Fatalf("expected dryRun output, got %q", stdout)
	}
}

func TestReleaseStatusRejectsInvalidPollInterval(t *testing.T) {
	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{
			"release", "status",
			"--package-name", "com.example.app",
			"--track", "production",
			"--wait",
			"--poll-interval", "0s",
		}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		err := root.Run(context.Background())
		if !errors.Is(err, flag.ErrHelp) {
			t.Fatalf("expected ErrHelp, got %v", err)
		}
	})

	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "Error: --poll-interval must be greater than 0") {
		t.Fatalf("expected poll-interval validation error, got %q", stderr)
	}
}

func TestAPKsUploadRequiresAPKExtension(t *testing.T) {
	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{
			"apks", "upload",
			"--package-name", "com.example.app",
			"--edit-id", "edit-1",
			"--apk", "release.zip",
		}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		err := root.Run(context.Background())
		if !errors.Is(err, flag.ErrHelp) {
			t.Fatalf("expected ErrHelp, got %v", err)
		}
	})

	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "Error: --apk must point to a .apk file") {
		t.Fatalf("expected apk extension error, got %q", stderr)
	}
}

func TestMetadataReleaseNotesSetRequiresLocale(t *testing.T) {
	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{
			"metadata", "release-notes", "set",
			"--package-name", "com.example.app",
			"--edit-id", "edit-1",
			"--track", "production",
			"--version-codes", "123",
			"--status", "completed",
			"--text", "Bug fixes",
		}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		err := root.Run(context.Background())
		if !errors.Is(err, flag.ErrHelp) {
			t.Fatalf("expected ErrHelp, got %v", err)
		}
	})

	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "Error: --locale is required") {
		t.Fatalf("expected locale validation error, got %q", stderr)
	}
}

func TestMetadataReleaseNotesSetRequiresConfirmProduction(t *testing.T) {
	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{
			"metadata", "release-notes", "set",
			"--package-name", "com.example.app",
			"--edit-id", "edit-1",
			"--track", "production",
			"--version-codes", "123",
			"--status", "completed",
			"--locale", "en-US",
			"--text", "Bug fixes",
		}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		err := root.Run(context.Background())
		if !errors.Is(err, flag.ErrHelp) {
			t.Fatalf("expected ErrHelp, got %v", err)
		}
	})

	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "Error: --confirm-production is required when --track is production") {
		t.Fatalf("expected confirm-production validation error, got %q", stderr)
	}
}

func TestListingsGetRequiresLanguage(t *testing.T) {
	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{
			"listings", "get",
			"--package-name", "com.example.app",
			"--edit-id", "edit-1",
		}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		err := root.Run(context.Background())
		if !errors.Is(err, flag.ErrHelp) {
			t.Fatalf("expected ErrHelp, got %v", err)
		}
	})

	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "Error: --language is required") {
		t.Fatalf("expected language validation error, got %q", stderr)
	}
}

func TestListingsUpdateRequiresContentField(t *testing.T) {
	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{
			"listings", "update",
			"--package-name", "com.example.app",
			"--edit-id", "edit-1",
			"--language", "en-US",
		}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		err := root.Run(context.Background())
		if !errors.Is(err, flag.ErrHelp) {
			t.Fatalf("expected ErrHelp, got %v", err)
		}
	})

	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "Error: at least one of --title, --short-description, --full-description, or --video is required") {
		t.Fatalf("expected content field validation error, got %q", stderr)
	}
}

func TestListingsUpdateRejectsInvalidLanguage(t *testing.T) {
	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{
			"listings", "update",
			"--package-name", "com.example.app",
			"--edit-id", "edit-1",
			"--language", "en_US",
			"--title", "New title",
		}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		err := root.Run(context.Background())
		if !errors.Is(err, flag.ErrHelp) {
			t.Fatalf("expected ErrHelp, got %v", err)
		}
	})

	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "Error: --language must be a valid BCP-47 tag") {
		t.Fatalf("expected language format validation error, got %q", stderr)
	}
}

func TestAppsListRequiresDeveloper(t *testing.T) {
	withTempConfigPath(t)
	t.Setenv("GOOGLE_PLAY_DEVELOPER", "")

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{"apps", "list"}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		err := root.Run(context.Background())
		if !errors.Is(err, flag.ErrHelp) {
			t.Fatalf("expected ErrHelp, got %v", err)
		}
	})

	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "Error: --developer is required") {
		t.Fatalf("expected developer validation error, got %q", stderr)
	}
}

func TestAppsListRejectsInvalidPageSize(t *testing.T) {
	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{
			"apps", "list",
			"--developer", "123456",
			"--page-size", "0",
		}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		err := root.Run(context.Background())
		if !errors.Is(err, flag.ErrHelp) {
			t.Fatalf("expected ErrHelp, got %v", err)
		}
	})

	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "Error: --page-size must be -1 or a positive integer") {
		t.Fatalf("expected page-size validation error, got %q", stderr)
	}
}

func TestConfigSetPersistsDefaults(t *testing.T) {
	configPath := withTempConfigPath(t)
	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{
			"config", "set",
			"--developer", "9092032418990165552",
			"--package-name", "com.zafarr.catbreedid",
		}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		if err := root.Run(context.Background()); err != nil {
			t.Fatalf("unexpected run error: %v", err)
		}
	})
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(stdout, `"scope":"global"`) {
		t.Fatalf("expected config output, got %q", stdout)
	}

	raw, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("ReadFile() error: %v", err)
	}
	var cfg map[string]any
	if err := json.Unmarshal(raw, &cfg); err != nil {
		t.Fatalf("Unmarshal() error: %v", err)
	}
	if cfg["default_developer"] != "9092032418990165552" {
		t.Fatalf("default_developer not persisted: %#v", cfg["default_developer"])
	}
	if cfg["default_package_name"] != "com.zafarr.catbreedid" {
		t.Fatalf("default_package_name not persisted: %#v", cfg["default_package_name"])
	}
}

func TestReleaseRunUsesConfiguredPackageDefault(t *testing.T) {
	configPath := withTempConfigPath(t)
	configJSON := `{"default_package_name":"com.example.fromconfig"}`
	if err := os.WriteFile(configPath, []byte(configJSON), 0o600); err != nil {
		t.Fatalf("WriteFile() error: %v", err)
	}

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{
			"release", "run",
			"--track", "production",
			"--status", "completed",
			"--version-codes", "123",
		}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		err := root.Run(context.Background())
		if !errors.Is(err, flag.ErrHelp) {
			t.Fatalf("expected ErrHelp, got %v", err)
		}
	})

	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "Error: --confirm-production is required when --track is production") {
		t.Fatalf("expected confirm-production validation error, got %q", stderr)
	}
}

func TestAppsListUsesConfiguredDeveloperDefault(t *testing.T) {
	configPath := withTempConfigPath(t)
	configJSON := `{"default_developer":"9092032418990165552"}`
	if err := os.WriteFile(configPath, []byte(configJSON), 0o600); err != nil {
		t.Fatalf("WriteFile() error: %v", err)
	}

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{
			"apps", "list",
			"--page-size", "0",
		}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		err := root.Run(context.Background())
		if !errors.Is(err, flag.ErrHelp) {
			t.Fatalf("expected ErrHelp, got %v", err)
		}
	})

	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "Error: --page-size must be -1 or a positive integer") {
		t.Fatalf("expected page-size validation error, got %q", stderr)
	}
}

func TestMissingCredentialsReturnsAuthExitCode(t *testing.T) {
	withTempConfigPath(t)
	t.Setenv("GOOGLE_PLAY_SERVICE_ACCOUNT_PATH", "")
	t.Setenv("GOOGLE_PLAY_SERVICE_ACCOUNT_JSON", "")
	t.Setenv("GOOGLE_PLAY_PROFILE", "")

	exitCode := Run([]string{"edits", "create", "--package-name", "com.example.app"}, "1.2.3")
	if exitCode != ExitAuth {
		t.Fatalf("expected exit code %d, got %d", ExitAuth, exitCode)
	}
}

func TestAuthDoctorMissingCredentialsReturnsAuthExitCode(t *testing.T) {
	withTempConfigPath(t)
	t.Setenv("GOOGLE_PLAY_SERVICE_ACCOUNT_PATH", "")
	t.Setenv("GOOGLE_PLAY_SERVICE_ACCOUNT_JSON", "")
	t.Setenv("GOOGLE_PLAY_PROFILE", "")

	exitCode := Run([]string{"auth", "doctor"}, "1.2.3")
	if exitCode != ExitAuth {
		t.Fatalf("expected exit code %d, got %d", ExitAuth, exitCode)
	}
}
