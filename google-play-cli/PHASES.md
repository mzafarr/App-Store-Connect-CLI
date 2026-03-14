# Google Play CLI Roadmap (Phase-by-Phase)

This document is the full execution roadmap for the standalone `google-play-cli`.

We will execute **one phase per chat**. Each phase is treated as a small release with clear scope, tests, and acceptance criteria.

## Operating model

- One phase per chat (strict scope control).
- Every phase starts with a short design note.
- Every phase ends with validation (`make format`, `make test`, `make lint`, `make build`).
- `progress.md` is updated after each completed task and roadblock.

## Phase status legend

- `COMPLETED`: implemented and validated.
- `IN_PROGRESS`: currently active.
- `PLANNED`: approved backlog, not started.
- `BLOCKED`: waiting on external dependency or decision.

## Phase 0 — Standalone project foundation (`COMPLETED`)

### Goal

Create a standalone Google Play CLI project (separate from ASC) with independent module/build/test surface.

### Delivered

- Separate module under `google-play-cli/`.
- Independent root command and exit codes.
- Base client/auth scaffolding.

### Exit criteria met

- ASC project restored to ASC-only behavior.
- `google-play-cli` builds and tests independently.

## Phase 1 — Core edit + track operations (`COMPLETED`)

### Goal

Support base Android Publisher edit workflow.

### Delivered

- `edits create`
- `edits commit`
- `tracks list`
- `tracks update`

### Exit criteria met

- Input validations + command tests.
- API operations verified by unit tests.

## Phase 2 — Artifact upload + release orchestration (`COMPLETED`)

### Goal

Support binary upload and one-command release flow.

### Delivered

- `bundles upload`
- `apks upload`
- `release run` (create edit -> upload/use version -> update track -> commit)

### Exit criteria met

- Upload endpoints covered by tests.
- Dry-run behavior validated.

## Phase 3 — Auth UX, profiles, and diagnostics (`COMPLETED`)

### Goal

Make authentication easier to debug and safer to operate.

### Delivered

- `auth doctor`
- Profile-aware config loading (`--profile`, `GOOGLE_PLAY_PROFILE`)
- Config file support (`~/.gplay/config.json`, `GPCLI_CONFIG_PATH`)

### Exit criteria met

- Credential source reporting works.
- Profile/env resolution tests pass.

## Phase 4 — Release visibility + metadata (`COMPLETED`)

### Goal

Track release state and manage localized changelogs.

### Delivered

- `release status` + `--wait`
- `metadata release-notes get`
- `metadata release-notes set`
- Fixes for API shape quirks (edits-based track reads, string versionCodes parsing)

### Exit criteria met

- Live status checks on real account succeeded.
- Regression tests added for parsing/endpoint behavior.

## Phase 5 — Safety + persisted defaults (`COMPLETED`)

### Goal

Prevent accidental production writes and avoid repetitive flags.

### Delivered

- Production safety gate:
  - `--confirm-production` required for production writes in:
    - `release run` (non-dry-run),
    - `tracks update`,
    - `metadata release-notes set`.
- Persisted defaults:
  - `config set`, `config show`
  - fallback order: flag -> env -> profile config -> global config
  - saved defaults for:
    - developer ID,
    - package name.
- Discovery:
  - `apps list` from `users.list` grants.

### Exit criteria met

- Runtime smoke checks pass.
- Live read operations work using saved defaults without repeating flags.

## Phase 6 — CI + release packaging (`COMPLETED`)

### Goal

Make the CLI releasable with reproducible artifacts.

### Delivered

- CI checks workflow for `google-play-cli` path changes.
- Release artifacts workflow:
  - multi-platform binaries/archives,
  - `checksums.txt`.
- Local dist target:
  - `make dist VERSION=vX.Y.Z`.

### Exit criteria met

- Build matrix and checksum generation validated.

---

## Phase 7 — Listings management (`COMPLETED`)

### Goal

Support app store listing text operations.

### Delivered

- `listings get`
- `listings list`
- `listings update`

### API scope

- `edits.listings.*`

### Delivered details

- Added `edits.listings.*` client coverage:
  - list,
  - get by language,
  - patch update.
- Added CLI command group and wiring:
  - `gplay listings list --edit-id ...`
  - `gplay listings get --edit-id ... --language ...`
  - `gplay listings update --edit-id ... --language ... [content flags]`
- Added validation:
  - required `--edit-id`,
  - required valid BCP-47 `--language`,
  - at least one content field for updates.
- Added RED->GREEN tests for command validation and endpoint/payload behavior.

### Exit criteria met

- Can list/read localized listings.
- Can patch localized listing title/short/full description/video safely.
- Full validation suite passes after implementation.

## Phase 8 — Images/assets management (`PLANNED`)

### Goal

Support screenshots/icon/feature graphic operations.

### Candidate commands

- `images list`
- `images upload`
- `images delete`

### API scope

- `edits.images.*`

### Deliverables

- Image type enum handling (phone/tablet/etc.).
- Upload + list + delete with robust file/path validation.

### Exit criteria

- End-to-end image lifecycle works in edit flow.

## Phase 9 — Testers and track audience control (`PLANNED`)

### Goal

Manage tester assignments and rollout audiences per track.

### Candidate commands

- `testers list`
- `testers update`

### API scope

- `edits.testers.*`

### Exit criteria

- Internal/closed test audiences manageable from CLI.

## Phase 10 — App details + country availability (`PLANNED`)

### Goal

Support app-level metadata and market availability controls.

### Candidate commands

- `details get/update`
- `country-availability get`

### API scope

- `edits.details.*`
- `edits.countryavailability.get`

### Exit criteria

- CLI can inspect/update key app details and read market availability.

## Phase 11 — Reviews operations (`PLANNED`)

### Goal

Enable support workflows from CLI.

### Candidate commands

- `reviews list`
- `reviews get`
- `reviews reply`

### API scope

- `reviews.*`

### Exit criteria

- Review triage and reply possible via CLI.

## Phase 12 — Monetization + purchase validation (`PLANNED`)

### Goal

Add operational commands for monetization and purchase verification.

### Candidate commands

- `purchases products get`
- `purchases subscriptionsv2 get`
- selected `orders` reads/refunds
- selected `monetization` reads

### API scope

- `purchases.*`
- `orders.*`
- `monetization.*`

### Exit criteria

- Can validate purchase tokens and inspect key monetization data from CLI.

## Phase 13 — Access management helpers (`PLANNED`)

### Goal

Make account/app access troubleshooting easier.

### Candidate commands

- `users list`
- `grants list` (derived from users payload)

### API scope

- `users.*`
- `grants.*`

### Exit criteria

- Permission diagnostics available without Play Console navigation.

## Phase 14 — UX polish + docs + stability (`PLANNED`)

### Goal

Harden UX and improve maintainability.

### Deliverables

- Error taxonomy cleanup and actionable messages.
- Help text consistency pass.
- Expanded examples for every command group.
- Final pass on integration-style smoke tests.

### Exit criteria

- New user can self-serve setup and first release from docs only.

---

## Per-phase Definition of Done

A phase is only complete when all are true:

1. Scope delivered (commands/flags/API behavior).
2. Tests added/updated and passing.
3. `make format && make test && make lint && make build` passes.
4. README/help updated for new user-facing behavior.
5. `progress.md` updated with outcomes and roadblocks.
