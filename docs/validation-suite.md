# Validation Suite

Purpose
- A scheduled, versioned validation suite exercising primary user journeys
  using deterministic, sanitized fixtures.

Goals
- Exercise transaction debug, replay/trace, profile export, session save, audit
  signing surface, protocol registration, and release metadata verification.
- Run against at least Linux and one additional platform (Windows).
- Produce sanitized artifacts and a machine-readable summary so failures map to
  the broken stage and include fingerprints for regression detection.

Running locally

Build the binary and run the suite locally:

```bash
go build -o bin/glassbox ./cmd/glassbox
npm ci --prefer-offline
npm run build --if-present
chmod +x scripts/run-validation-suite.sh
GLASSBOX_BINARY=./bin/glassbox scripts/run-validation-suite.sh
```

CI Integration

- A GitHub Actions workflow `.github/workflows/validation-suite.yml` runs the
  suite quarterly on `ubuntu-latest` and `windows-latest` and uploads artifacts
  under `validation-<os>-<run-id>` with a 30-day retention. Artifacts are
  redacted via `scripts/redact-logs.sh` before upload to remove test-only
  secrets and PEM blocks.

Reporting & Retention

- Per-run artifacts are retained by GitHub Actions for 30 days (configured in
  the workflow). The validation runner writes `ci-artifacts/validation/summary.json`
  and `report.md` which are uploaded as part of the artifact bundle.

- A helper script `scripts/generate-compat-report.sh` aggregates multiple
  `summary.json` files into `compatibility_report.md` to support quarterly
  compatibility reviews and follow-up issue tracking.

- Long-term retention of specific failing-run artifacts should be performed by
  maintainers downloading the artifact and storing it in an internal archive
  if required for extended investigations. By default, the CI retention policy
  is intentionally short to limit exposure of debugging data.

Mutation testing subsystem selection

The mutation testing targets in the `Makefile` support running a single
subsystem or all subsystems. Subsystem selection is driven by environment
variables so the same targets work locally and in CI.

- `MUTATION_SUBSYSTEMS` is a space-separated list of subsystem names. Each
  name selects a group of packages and a mutation score threshold.
- For each subsystem `<name>` in `MUTATION_SUBSYSTEMS`, the following variables
  are read:
  - `MUTATION_SUBSYSTEM_<name>_PACKAGES`: the Go package pattern(s) to mutate.
  - `MUTATION_SUBSYSTEM_<name>_THRESHOLD`: the minimum mutation score (percent)
    required for that subsystem.
  - `MUTATION_SUBSYSTEM_<name>_EXCLUDE`: optional comma-separated list of
    packages or files to exclude from mutation.
- `MUTATION_SUBSYSTEM` selects a single subsystem for
  `make mutation-test-ci-subsystems` and must be set to one of the names in
  `MUTATION_SUBSYSTEMS`. If it is unset or not a known subsystem, the target
  prints an error and exits non-zero.
- `MUTATION_REPORT_DIR` sets the directory where per-subsystem reports are
  written. Reports are laid out as
  `$MUTATION_REPORT_DIR/<subsystem>/mutation-report.json` (plus any summary
  files produced by the mutation tool).

Run all subsystems (success path):

```bash
make mutation-test-subsystems
```

This iterates over `MUTATION_SUBSYSTEMS`, runs mutation testing for each
subsystem's packages, and writes reports under `MUTATION_REPORT_DIR`. The
target exits `0` only when every subsystem meets its threshold.

Run a single subsystem in CI (failure path when the selection is invalid):

```bash
# Valid: runs the subsystem named in MUTATION_SUBSYSTEM
MUTATION_SUBSYSTEM=core make mutation-test-ci-subsystems

# Invalid: unknown subsystem name
MUTATION_SUBSYSTEM=does-not-exist make mutation-test-ci-subsystems
```

The invalid invocation prints an error identifying the unknown subsystem and
exits non-zero (the same happens when `MUTATION_SUBSYSTEM` is unset). Record
the exact error text and exit code from the current `Makefile` when updating
this section.

Fixtures

- Deterministic fixtures live under `test/validation/fixtures/` and must avoid
  any real keys, hashes, or network credentials. Audit fixtures must use the
  `testonly_` filename prefix.

Acceptance checklist

- Every journey has a deterministic fixture and documented prerequisites.
- Failures produce per-journey logs and a consolidated `report.md` describing
  exit codes and fingerprints.
- The workflow runs quarterly and covers Linux + Windows.
- Artifacts are redacted via `scripts/redact-logs.sh` before upload and
  retained per CI policy.
