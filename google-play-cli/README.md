# Google Play CLI (Standalone)

A standalone CLI for Google Play Android Publisher workflows, inspired by the App Store Connect CLI structure.

## Planning and execution docs

- Full phase roadmap: `PHASES.md`
- Ongoing execution log: `progress.md`

## Required inputs and where to get them

| Input | Why it is needed | Where to get it |
| --- | --- | --- |
| Service account JSON key | Authenticate API calls | Google Cloud Console -> IAM & Admin -> Service Accounts -> Keys -> Create key (JSON) |
| Package name (`com.example.app`) | Target app for edits/releases | App `applicationId`, or Play Console app details |
| Developer ID (`9092032418990165552`) | Needed for `gplay apps list` | Play Console URL: `.../developers/<DEVELOPER_ID>/...` |
| Edit ID (`EDIT_ID`) | Required by edit-scoped commands (tracks/listings updates) | Output of `gplay edits create` or `release run` response |
| Listing language (`en-US`) | Required by `gplay listings get/update` | Play Console listing locale / BCP-47 language tag |
| Version code (`7`, `123`) | Required when releasing existing artifacts | `gplay release status --track <track>` output, or your build metadata |

## Save defaults once (no repeated flags)

Persist values locally so commands can omit repeated flags:

```bash
gplay config set --developer 9092032418990165552 --package-name com.zafarr.catbreedid
gplay config show --pretty
```

After this:

- `gplay apps list` can use saved developer ID.
- Commands that need `--package-name` can use saved package automatically.

## First-time setup (Google Cloud + Play Console)

1. Open Google Cloud Console and select the project you use for Play Console API access.
2. Enable the **Google Play Android Developer API** for that project.
3. Go to **IAM & Admin → Service Accounts** and create (or select) a service account.
4. Create a key: **Keys → Add key → Create new key → JSON**.
5. In Google Play Console, open **Setup → API access**, link the same Cloud project, and grant that service account permissions for your app.

Then configure credentials for the CLI:

```bash
export GOOGLE_PLAY_SERVICE_ACCOUNT_PATH="/absolute/path/to/service-account.json"
```

Alternative:

```bash
export GOOGLE_PLAY_SERVICE_ACCOUNT_JSON='{"type":"service_account", ...}'
```

Security note: treat service-account JSON as a secret, do not commit it, and keep file permissions restricted.

## Quick test on a real account (safe flow)

Use `internal` + `draft` for testing to avoid accidental production rollout.

```bash
cd google-play-cli
make build

./gplay auth doctor --pretty
./gplay config set --developer <DEVELOPER_ID> --package-name com.your.app
./gplay release status --track internal --pretty
./gplay apps list --pretty

# Safe dry run (no API mutations)
./gplay release run --dry-run \
  --track internal \
  --status draft \
  --version-codes <EXISTING_VERSION_CODE> \
  --pretty

# Optional real mutation on internal track (still draft)
./gplay release run \
  --track internal \
  --status draft \
  --version-codes <EXISTING_VERSION_CODE> \
  --pretty

# Listings flow
EDIT_ID="$(./gplay edits create --output json | jq -r .id)"
./gplay listings list --edit-id "$EDIT_ID" --pretty
./gplay listings get --edit-id "$EDIT_ID" --language en-US --pretty
./gplay listings update --edit-id "$EDIT_ID" --language en-US --short-description "Improved ML cat matching" --pretty
./gplay edits commit --edit-id "$EDIT_ID" --pretty
```

Do not use `production` unless you intentionally want to ship. Production-track mutations require `--confirm-production`.

## Troubleshooting

- `zsh: no such file or directory: ./gplay`  
  Run `make build` inside `google-play-cli` first.

- `Package not found: <package>`  
  Verify the exact package name and ensure the service account has access to that app in Play Console (`Setup -> API access`).

- `--confirm-production is required when --track is production`  
  This is a safety gate. Re-run with `--confirm-production` only when you intentionally want a production-track write.

- Auth errors or missing credentials  
  Run `./gplay auth doctor --pretty` and confirm `GOOGLE_PLAY_SERVICE_ACCOUNT_PATH` or profile config is set correctly.

- Unsure about package IDs  
  Use `./gplay apps list --developer <DEVELOPER_ID> --pretty` to list package names visible through account grants.

## Current commands

- `gplay auth doctor`
- `gplay config set --developer "1234567890123456789" --package-name "com.example.app"`
- `gplay config show --pretty`
- `gplay apps list [--developer "1234567890123456789"]`
- `gplay listings list [--package-name "com.example.app"] --edit-id "EDIT_ID"`
- `gplay listings get [--package-name "com.example.app"] --edit-id "EDIT_ID" --language "en-US"`
- `gplay listings update [--package-name "com.example.app"] --edit-id "EDIT_ID" --language "en-US" --title "New title"`
- `gplay edits create [--package-name "com.example.app"]`
- `gplay edits commit [--package-name "com.example.app"] --edit-id "EDIT_ID"`
- `gplay tracks list [--package-name "com.example.app"] --edit-id "EDIT_ID"`
- `gplay tracks update [--package-name "com.example.app"] --edit-id "EDIT_ID" --track "production" --version-codes "123" --status completed --confirm-production`
- `gplay bundles upload [--package-name "com.example.app"] --edit-id "EDIT_ID" --aab "./app-release.aab"`
- `gplay apks upload [--package-name "com.example.app"] --edit-id "EDIT_ID" --apk "./app-release.apk"`
- `gplay release run [--package-name "com.example.app"] --track "production" --status completed --aab "./app-release.aab" --confirm-production [--wait]`
- `gplay release status [--package-name "com.example.app"] --track "production" [--wait]`
- `gplay metadata release-notes get [--package-name "com.example.app"] --track "production"`
- `gplay metadata release-notes set [--package-name "com.example.app"] --edit-id "EDIT_ID" --track "production" --version-codes "123" --status completed --locale en-US --text "Bug fixes" --confirm-production`

## Authentication

Set one of:

- `GOOGLE_PLAY_SERVICE_ACCOUNT_PATH=/absolute/path/to/service-account.json`
- `GOOGLE_PLAY_SERVICE_ACCOUNT_JSON='{"type":"service_account", ...}'`

Optional profile/config support:

- Root flag: `--profile`
- Env: `GOOGLE_PLAY_PROFILE`
- Env: `GOOGLE_PLAY_PACKAGE_NAME` (default package fallback)
- Env: `GOOGLE_PLAY_DEVELOPER` (default developer fallback)
- Config path env override: `GPCLI_CONFIG_PATH`
- Default config file: `~/.gplay/config.json`

Example config:

```json
{
  "default_profile": "prod",
  "default_developer": "9092032418990165552",
  "default_package_name": "com.zafarr.catbreedid",
  "profiles": {
    "prod": {
      "service_account_path": "/Users/me/.keys/play-prod.json",
      "developer": "9092032418990165552",
      "package_name": "com.zafarr.catbreedid"
    },
    "staging": {
      "service_account_path": "/Users/me/.keys/play-staging.json",
      "developer": "9092032418990165552",
      "package_name": "com.zafarr.catbreedid.staging"
    }
  }
}
```

Optional overrides:

- `GOOGLE_PLAY_TOKEN_URL`
- `GOOGLE_PLAY_API_BASE_URL`
- `GOOGLE_PLAY_TIMEOUT`
- `GOOGLE_PLAY_TIMEOUT_SECONDS`

## Developer account ID for `apps list`

Use the numeric developer ID from your Play Console URL, for example:

- URL: `https://play.google.com/console/developers/1234567890123456789/app-list`
- Developer ID: `1234567890123456789`

## Build and test

```bash
cd google-play-cli
make format
make test
make build
make dist VERSION=v0.1.0
```

## Release artifacts

CI release packaging workflow: `.github/workflows/google-play-cli-release.yml`

- Builds: `linux`/`darwin` (`amd64`, `arm64`) and `windows` (`amd64`)
- Outputs archives + `checksums.txt` as workflow artifacts
- Trigger via tag: `gplay-v*` or manual `workflow_dispatch`
