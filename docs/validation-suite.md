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

- **`summary.json` schema** — each file is a JSON object with a `journeys` array.
  Every journey entry has the following fields:

  | Field | Type | Description |
  |-------|------|-------------|
  | `name` | string | Journey identifier (e.g. `debug-trace`, `replay-deterministic`) |
  | `exitcode` | int | Exit code of the journey script (0 = pass) |
  | `fingerprint` | string | 12-char hex derived from name + exitcode + fixture hash |
  | `output` | string | Tail of journey log (last 50 lines, redacted) |

  Example:
  ```json
  {
    "journeys": [
      { "name": "debug-trace",          "exitcode": 0, "fingerprint": "a3f8c12d4e5b", "output": "..." },
      { "name": "replay-deterministic", "exitcode": 0, "fingerprint": "b7e1d09f3c2a", "output": "..." }
    ]
  }
  ```

- A helper script `scripts/generate-compat-report.sh` aggregates multiple
  `summary.json` files into `compatibility_report.md` to support quarterly
  compatibility reviews and follow-up issue tracking. **Note:** this script
  is not yet present in the repository; until it is added, aggregate the
  `summary.json` files manually using `jq '.journeys[]' summary.json`.

- Long-term retention of specific failing-run artifacts should be performed by
  maintainers downloading the artifact and storing it in an internal archive
  if required for extended investigations. By default, the CI retention policy
  is intentionally short to limit exposure of debugging data.

Artifact redaction

- Before upload, all artifact files are passed through `scripts/redact-logs.sh`
  which removes tokens, PEM blocks, API keys, and connection strings.
  **Note:** `scripts/redact-logs.sh` is not yet present in the repository;
  until it is added, the upload step in `validation-suite.yml` skips redaction
  (`|| true`) and logs a warning. Do not share raw artifacts externally.

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
