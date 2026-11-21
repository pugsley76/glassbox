# Regression Test Fixture Guide

This document defines the naming conventions, directory layout, minimal
reproduction rules, and secret-avoidance requirements for all regression test
fixtures in `test/regression/fixtures/`.

---

## Directory layout

```
test/regression/
├── FIXTURES.md              ← you are here
├── fixture_index.json       ← auto-generated index (see scripts/index_regression_fixtures.sh)
├── fixtures/
│   ├── rpc/                 ← Stellar RPC response stubs (JSON)
│   ├── replay/              ← Snapshot registries and ledger-state files
│   ├── trace/               ← Serialised ExecutionTrace JSON files
│   ├── sourcemap/           ← Minimal WASM binaries and alias JSON files
│   ├── session/             ← Session record JSON and SQLite dumps
│   ├── audit/               ← Audit payloads, signed logs, and TEST-ONLY keys
│   └── cli/                 ← Expected CLI output fragments and env overrides
```

Each subdirectory has its own `README.md` with layer-specific rules.

Text fixtures must use LF line endings; see [Line endings](#line-endings) for
the rule and the check command.

### Fixture index

`fixture_index.json` is a deterministic, auto-generated catalogue of every
fixture file.  Regenerate it after adding or removing fixtures:

```bash
./scripts/index_regression_fixtures.sh
```

Run validation without regenerating (e.g. in CI):

```bash
./scripts/index_regression_fixtures.sh --check-only
```

The index records each fixture's `path`, `layer`, `failure_class`, `issue_ref`,
`schema_version`, and `test_name`.  The script exits non-zero if any fixture
lacks a valid layer, lacks an issue/PR reference in its filename, or shares a
fixture ID with another file in its layer. See
[Naming convention](#naming-convention) for the exact rules and
[Known baseline](#known-baseline) for the current state of `main`.

---

## Canonical stub values

These values are used consistently across all fixtures to prevent accidental
use of real network data.

| Field              | Stub value                                                         |
|--------------------|--------------------------------------------------------------------|
| Transaction hash   | `5c0a1234567890abcdef1234567890abcdef1234567890abcdef1234567890ab` |
| Envelope XDR       | `AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=`                     |
| Network            | `testnet`                                                          |
| Timestamp          | `2026-01-01T00:00:00Z`                                             |
| Session ID prefix  | `sess_test_`                                                       |
| Contract ID prefix | `CTEST`                                                            |

---

## Naming convention

Every fixture file follows:

```
<layer>_<scenario-slug>_<issue-or-pr-slug>.<ext>
```

Where:

- `<layer>` — one of: `rpc`, `replay`, `trace`, `sourcemap`, `session`,
  `audit`, `cli`. It must match the subdirectory the file lives in.
- `<scenario-slug>` — short snake_case description of the failure class being
  reproduced (e.g. `budget_exhaustion`, `missing_footprint`, `empty_events`)
- `<issue-or-pr-slug>` — **required**. Write it as `issue<N>` or `pr<N>`, where
  `<N>` is the GitHub number in digits with no separator (e.g. `issue234`,
  `pr401`).
- `<ext>` — file extension appropriate for the content type. Compound
  extensions such as `.trace.json` or `.expected.txt` are allowed.

Line endings are not part of the name, but they are checked: text fixtures use
LF only. See [Line endings](#line-endings).

### How the index script applies these rules

`scripts/index_regression_fixtures.sh` is the only automated check on fixture
names. This is what it actually enforces:

| Rule | Enforced? | Detail |
|------|-----------|--------|
| File lives in a valid layer directory | Yes (fail) | Other directories are skipped with a `[WARN]` |
| Issue/PR reference present | Yes (fail) | Matched by the regex `(?:issue\|pr)(\d+)`, case-insensitive. A missing reference prints `[WARN]` but **counts as a failure** |
| Unique fixture ID per layer | Yes (fail) | The ID is the filename with **every** extension removed. `foo_issue1.env.json` and `foo_issue1.expected.txt` both become `foo_issue1` and collide |
| `<layer>_` filename prefix | No | Not checked. Use the prefix anyway |
| `README.md` | Exempt | Ignored in every layer directory |

Things this implies:

- `pr_401`, `pr_offline`, and `issue-234` do **not** match the regex. Leave
  out the separator: `pr401`, `issue234`.
- A fixture with no GitHub number cannot pass the check. Use the number of the
  issue or PR that adds the fixture.
- Companion files for the same scenario (input plus expected output) need
  distinct stems, for example `cli_format_yaml_issue230_env.json` and
  `cli_format_yaml_issue230_expected.txt`.

Examples:

| Filename | Result | Reason |
|----------|--------|--------|
| `rpc_gettransaction_notfound_issue150.json` | pass | Layer prefix, slug, `issue150` |
| `trace_empty_steps_pr319.trace.json` | pass | `pr319` matches; compound extension is fine |
| `audit_empty_payload_rejected.payload.json` | **fail** | No issue/PR reference |
| `rpc_sendtransaction_internal_error_pr_offline.json` | **fail** | `pr_offline` has no digits |
| `cli_format_yaml_rejected_issue230.env.json` + `cli_format_yaml_rejected_issue230.expected.txt` | **fail** | Both reduce to the ID `cli_format_yaml_rejected_issue230` |
| `gethealth_truncated_missing_status.json` | **fail** | No reference (and no `rpc_` prefix, which is not checked) |

Every **fail** row is a file that is currently checked in. See
[Known baseline](#known-baseline).

### Checking names

From the repository root (requires `bash` and `python3`):

```bash
./scripts/index_regression_fixtures.sh --check-only
```

This prints `[OK]   All fixture index checks passed.` and exits 0 when
everything passes. When anything fails it prints one
`[WARN]`/`[FAIL]` line per problem and a summary such as `[FAIL] 45 check(s) failed.`,
then exits 1.

### Known baseline

On current `main`, `--check-only` exits 1 with `45 check(s) failed.` This
failure is pre-existing and does not come from any single new fixture. It
breaks down as:

- 15 fixtures have no issue/PR reference: all 13 `rpc/` RPC-shape fixtures
  (`gethealth_*`, `getlatestledger_*`, `getledgerentries_*`,
  `simulatetransaction_*`, `error_object_empty_message.json`),
  `rpc_sendtransaction_internal_error_pr_offline.json`, and
  `audit_empty_payload_rejected.payload.json`.
- 1 duplicate ID: the two `cli_format_yaml_rejected_issue230.*` files.
- 29 index mismatches. The committed `fixture_index.json` is hand-written and
  out of sync with the script. It lists 8 of 21 fixtures, uses
  `fixtures/...` paths where the script emits `test/regression/fixtures/...`,
  stores Go test function names in `test_name` (the script stores the
  filename stem), and has no `schema_version`. Regenerating with the script
  would replace those `test_name` values.

When you add a fixture, compare the failure count before and after your
change. Your fixture should add no new `[WARN]` or `[FAIL]` lines.

---

## Line endings

Fixtures are committed and checked out on Linux, macOS, and Windows, so line
endings are part of the fixture contract:

- **Text fixtures use LF only.** This covers every `.json`, `.txt`, `.env`,
  `.md`, and shell (`.sh`) fixture under `test/regression/fixtures/`. A CRLF
  byte pair in a text fixture changes the fixture's byte content and can make
  golden-output comparisons fail on Linux CI.
- **CRLF is only permitted for fixtures that are explicitly Windows or
  binary.** Such a fixture must say so in its filename or in its layer
  `README.md` (for example a `.crlf.txt` fixture that exercises Windows line
  handling). Binary fixtures (e.g. `.wasm`) are not line-ending checked.

Do not rely on `core.autocrlf` or `.gitattributes` to normalise fixtures after
the fact; write the fixture with LF from the start.

### Checking line endings

From the repository root (requires `bash`):

```bash
./scripts/validate-docs.sh --fixtures
```

This runs the focused CRLF check over `test/regression/fixtures/`. On success
it prints:

```
[OK]   Fixture line endings are LF-only.
```

and exits 0. When a text fixture contains CRLF it prints one line per file and
exits 1, for example:

```
[FAIL] CRLF line endings in test/regression/fixtures/cli/cli_format_yaml_issue230_expected.txt
[FAIL] 1 fixture(s) with CRLF line endings.
```

Fix the reported file by rewriting it with LF endings (for example
`sed -i 's/\r$//' <file>` on Linux/macOS) and re-run the check.

---

## Minimal reproduction rules

A fixture must demonstrate **the smallest possible input that triggers the
specific failure class** being tested.  Specifically:

1. **Include only the fields required to hit the code path under test.**
   Leave all other fields as zero-values or omit them.

2. **Use the canonical stub values above** unless the test explicitly exercises
   field validation (e.g. testing that an empty transaction hash is rejected).

3. **One fixture per failure class.**  If two bugs share an input shape but
   differ in which code path fails, create two separate fixtures with distinct
   scenario slugs.

4. **Document the original failure** in a comment at the top of the fixture
   file (JSON `_comment` key, shell `#` comment, or a companion `.md` file):

   ```json
   {
     "_comment": "Reproduces issue #234: GetTransaction returns NOT_FOUND but the CLI printed 'success'",
     "status": "NOT_FOUND",
     ...
   }
   ```

---

## Secret avoidance

Fixtures must **never** contain:

- Real private keys, PKCS#8 PEM blobs, HSM PINs, or AWS KMS key ARNs
- Real Stellar transaction hashes from mainnet or testnet
- Real contract IDs from deployed production contracts
- Bearer tokens, API keys, or RPC authentication credentials
- Personal data or internal infrastructure hostnames

Keys placed in `test/regression/fixtures/audit/` are **test-only** and must be
labeled `testonly_` in both the filename and the file content.

If a fixture accidentally contains sensitive data, rotate the affected
credential immediately and remove the fixture from git history before the
branch is merged.

---

## Linking a fixture to the issue workflow

When filing a GitHub issue that requires a regression test, include the
following section in the issue body:

```markdown
## Regression fixture

- **Fixture directory**: `test/regression/fixtures/<layer>/`
- **Suggested filename**: `<layer>_<scenario>_issue<N>.<ext>` (digits only after `issue`/`pr`, no separator)
- **Failure class**: <one-line description of the class>
- **Canonical input**: paste a minimal JSON / shell snippet here
```

The fix PR must include both the fixture file and a test that loads it via the
helpers in `internal/testhelpers/`.  See
[docs/regression-test-guide.md](../../docs/regression-test-guide.md) for the
full contributor workflow.

---

## Regenerating fixtures

Some fixtures (particularly signed audit logs) must be regenerated when the
canonicalization algorithm or schema version changes.  A `# regenerate:`
comment in the fixture file records the exact command used:

```json
{
  "_regenerate": "glassbox audit:sign --payload-file fixtures/audit/canonical_hash_deterministic.payload.json --software-private-key fixtures/audit/testonly_ed25519.key.pem"
}
```

Run that command and replace the fixture with the new output.  Commit both
files together so the fixture and its generator stay in sync.
