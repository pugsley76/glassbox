# CI Artifacts Guide

This document explains what the CI pipeline collects when a job fails, how to
retrieve artifacts, what is redacted before upload, and how to reproduce a
failure locally using the downloaded material.

---

## When are artifacts uploaded?

**Failure-debug artifacts** are uploaded only when a job fails. Successful
runs do not upload anything beyond what is already visible in the job log.

**Mutation test reports** are uploaded on every mutation-test run (pass or
fail) so maintainers can track surviving-mutant trends over time.

---

## Artifact names and contents

| Artifact name | Uploaded when | Contents | Retention |
|---|---|---|---|
| `failure-unit-<os>-<run-id>` | `go-unit` job fails | JUnit XML, test JSON output, coverage profile, any `.trace.json` files written during the run | 7 days |
| `failure-integration-<run-id>` | `go-integration` job fails | JUnit XML, test JSON output, `glassbox-version.json`, `.trace.json` and `.registry.json` files | 7 days |
| `failure-ts-<run-id>` | `ts-test` job fails | JUnit XML, Jest JSON results, Jest console log, Node/npm version file | 7 days |
| `failure-regression-<run-id>` | `go-regression` job fails | JUnit XML, test JSON output, `.trace.json` files from regression runs | 7 days |
| `mutation-report-<run-id>` | Every mutation-test run | `gremlins-report.json`, `gremlins.log`, `summary.txt` | 30 days |

### Why 7 days for failure artifacts?

Seven days is long enough for the responsible engineer to download and inspect
the material before the next release cycle. Keeping failure artifacts
indefinitely risks retaining sensitive debugging information past its useful
lifetime. If you need longer retention for a specific investigation, download
the artifact locally immediately.

---

## Secret redaction

Every file in a failure-debug artifact staging directory is passed through
`scripts/redact-logs.sh` before upload. The following patterns are replaced
with `[REDACTED]`:

| Category | Examples matched |
|---|---|
| Authorization headers | `Authorization: Bearer <token>` |
| GitHub tokens | `ghp_...`, `ghs_...`, `gho_...`, `github_pat_...` |
| Generic API keys | `api_key=...`, `apikey: ...` |
| Passwords | `password=...`, `passwd=...`, `passphrase=...` |
| PEM private key blocks | `-----BEGIN PRIVATE KEY-----` ... `-----END PRIVATE KEY-----` |
| AWS credentials | `AKIA...`, `ASIA...`, `aws_secret_access_key=...` |
| PKCS#11 PINs | `pkcs11_pin=...`, `GLASSBOX_PKCS11_PIN=...` |
| RPC tokens | `rpc_token=...`, `GLASSBOX_RPC_TOKEN=...` |
| Connection strings | `://user:password@host` |
| Sentry DSN URLs | `https://<key>@sentry.io/...` |
| Hex private keys | Long hex strings in `secret_key=` / `private_key=` contexts |

The redaction script also enforces a **50 MB** size limit. If the staging
directory exceeds this limit the script exits with code 1 and the upload step
is skipped — the job log will show a warning. Reduce the number of collected
files or raise `ARTIFACT_MAX_MB` in `ci-test.yml` if this limit is hit.

Redaction is best-effort. Do not assume an artifact is completely secret-free.
Never share raw artifact downloads with untrusted parties.

---

## How to retrieve an artifact

### GitHub Actions UI

1. Go to the **Actions** tab of the repository.
2. Click the failed workflow run.
3. Scroll to the **Artifacts** section at the bottom of the summary page.
4. Click the artifact name to download a `.zip` file.

### GitHub CLI

```bash
# List artifacts for a specific run (replace RUN_ID with the numeric run ID
# visible in the Actions URL, e.g. 12345678901)
gh run download RUN_ID --repo chazepay/glassbox

# Download a specific artifact by name
gh run download RUN_ID --name failure-unit-ubuntu-latest-RUN_ID \
  --repo chazepay/glassbox --dir ./downloaded-artifacts
```

### GitHub REST API

```bash
# List artifacts for a run
curl -H "Authorization: Bearer $GITHUB_TOKEN" \
  "https://api.github.com/repos/chazepay/glassbox/actions/runs/RUN_ID/artifacts"

# Download (replace ARTIFACT_ID with the id from the list response)
curl -L -H "Authorization: Bearer $GITHUB_TOKEN" \
  "https://api.github.com/repos/chazepay/glassbox/actions/artifacts/ARTIFACT_ID/zip" \
  -o artifact.zip
unzip artifact.zip -d ./downloaded-artifacts
```

---

## Reproducing a failure locally

### Go unit / regression test failure

1. Download and extract the failure artifact.
2. Open `junit.xml` to identify the failing test name (e.g.
   `TestRegression_Session_MissingTxHash_ValidationFails`).
3. Run the specific test locally:

```bash
go test -v -run TestRegression_Session_MissingTxHash_ValidationFails \
  ./internal/cmd/...
```

4. If a `.trace.json` file is present in the artifact, replay it:

```bash
glassbox debug --load-snapshots ./path/to/snapshot.registry.json
```

5. Check `test-output.json` for the full structured test log — this includes
   timing, stdout/stderr per test, and the exit code.

### Integration test failure

1. Note the `glassbox-version.json` in the artifact — it records the exact
   binary version used in CI. Build the same version locally:

```bash
git checkout <commit-sha-from-version-json>
go build -o bin/glassbox ./cmd/glassbox
```

2. Run the integration tests against the locally built binary:

```bash
GLASSBOX_BINARY=./bin/glassbox \
  go test -v -timeout 120s ./integration/...
```

3. Trace files in the artifact can be diffed against a fresh local run:

```bash
# Export a fresh trace for the same transaction
glassbox debug --network testnet --trace-output fresh.trace.json <tx-hash>

# Diff the two JSON files (jq for pretty-printing)
diff <(jq . fresh.trace.json) <(jq . ci-trace.json)
```

### TypeScript test failure

1. Extract `jest-results.json` and open it to find the failing test suite and
   assertion.
2. Run the specific test file locally:

```bash
npx jest --testPathPattern="<path-to-test-file>" --verbose
```

3. `jest.log` in the artifact contains the full console output including any
   uncaught errors or module resolution failures.

---

## Mutation test report retrieval

1. Download `mutation-report-<run-id>` from the Actions UI or CLI:

```bash
gh run download RUN_ID --name mutation-report-RUN_ID \
  --repo chazepay/glassbox --dir ./mutation-report
```

2. Review the GitHub job summary for the `mutation-test` workflow — it
   contains a formatted table of surviving mutants grouped by package, posted
   automatically by the workflow.

3. For the full list, open `mutation-report/gremlins-report.json`:

```bash
# Pretty-print surviving mutants only
jq '[.mutants[] | select(.status == "SURVIVED")]' \
  mutation-report/gremlins-report.json
```

4. Run the same mutation test locally to reproduce:

```bash
# Install gremlins at the pinned version
make mutation-test-install

# Run mutation tests against a specific package
make mutation-test MUTATION_PACKAGES=./internal/session/...

# View the human-readable summary
make mutation-test-report
```

---

## Handling surviving mutants

When the mutation score falls below the agreed threshold (70 %), the
`mutation-test` job fails. To resolve it:

1. Review each surviving mutant in the report.
2. For **meaningful** survivors (the mutant changes observable behaviour that
   is not covered by any test): add a test that kills the mutant.
3. For **noise** survivors (the mutant is semantically equivalent to the
   original, e.g. changing a `>=` to `>` in a dead branch): add a
   `// mutation:skip` comment to the affected line with a one-sentence
   justification. Gremlins respects this directive and excludes the line from
   scoring.
4. Re-run `make mutation-test` locally to confirm the score improves, then
   push — the CI workflow will re-run automatically on the next push/PR event.

---

## Adjusting thresholds and scope

| Setting | Where to change | Effect |
|---|---|---|
| Mutation score threshold | `MUTATION_THRESHOLD` in `Makefile` and `mutation-test.yml` env | Jobs fail below this percentage |
| Package scope | `MUTATION_PACKAGES` in `Makefile` | Packages included in mutation runs |
| Artifact retention (failure) | `ARTIFACT_RETENTION_DAYS` in `ci-test.yml` env | How long failure artifacts are kept |
| Artifact size limit | `ARTIFACT_MAX_MB` in `ci-test.yml` env | Max total size before upload is blocked |
| Gremlins version | `GREMLINS_VERSION` in `Makefile` and `mutation-test.yml` env | Pinned tool version |

Changes to these settings should be reviewed and approved by a maintainer
because they affect the reliability signal visible to all contributors.


---

## SBOM diff and verification

Every release produces an SPDX 2.3 SBOM (`glassbox-<version>.spdx.json`). Three
scripts work with SBOMs in CI and locally:

- `cmd/generate-sbom` / `scripts/generate-sbom.sh` — generate the SBOM from the
  Go, Rust, and Node lockfiles.
- `scripts/sbom-diff.sh` — compare two SBOMs and report additions, removals,
  version changes, license changes, and policy violations.
- `scripts/verify-sbom.sh` — validate a single SBOM's structure and, with
  `--manifest`, confirm it is covered by the signed release manifest.

### Comparing two releases

```bash
# Compare the previous release SBOM against the current one (Make target)
make sbom-diff PREV_SBOM=dist/old/glassbox-v1.0.0.spdx.json \
               NEW_SBOM=dist/release/glassbox-v1.1.0.spdx.json

# Or call the script directly
bash scripts/sbom-diff.sh old.spdx.json new.spdx.json --policy license-policy.json
```

Text output (the default) lists each category with a count and the affected
components:

```
SBOM diff
  Old : old.spdx.json
  New : new.spdx.json
  Policy: license-policy.json

  Added:          1
  Removed:        1
  Version changes:2
  License changes:0
  Policy violations: 0
  New components (1):
    + github.com/new/dep v1.0.0 (go) [MIT]
  Removed components (1):
    - github.com/old/dep v1.0.0 (go) [MIT]
  Version changes (2):
    ~ github.com/shared/lib (go): v1.5.0 -> v1.6.0
```

For automation, request JSON or Markdown and write it to a file:

```bash
bash scripts/sbom-diff.sh old.spdx.json new.spdx.json \
  --format json --output sbom-diff.json
```

### Exit codes

| Code | Meaning |
|------|---------|
| 0 | No policy violations; additions, removals, and changes are advisory only |
| 1 | A prohibited or undeclared license was found in a new or changed component |
| 2 | A `--fail-on-*` threshold was exceeded (e.g. `--strict`) |

By default only policy violations fail the run. Use `--fail-on-addition`,
`--fail-on-removal`, `--fail-on-license-change`, or `--strict` (all three) to
fail on structural changes as well.

### Interpreting version changes

The diff keys components on their full PURL. The Glassbox SBOM generator embeds
the version in each PURL (e.g. `pkg:golang/github.com/example/lib@v1.2.3`), so a
dependency upgrade normally appears as a **removal** of the old version plus an
**addition** of the new version, not as a "version change". The "Version changes"
category fires only when a component keeps an identical PURL while its
`versionInfo` changes. When reviewing a diff between two Glassbox release SBOMs,
treat a paired removal + addition of the same package name as a version bump.

### Verifying a single SBOM

```bash
# Structural check plus manifest coverage
bash scripts/verify-sbom.sh glassbox-v1.1.0.spdx.json --manifest manifest.json
```

`verify-sbom.sh` checks SPDX-2.3 structure, PURL ecosystem coverage, and (with
`--manifest`) that the SBOM file is listed in the signed manifest with a matching
SHA-256. See [release-manifest.md](./release-manifest.md) for the manifest format
and [license-policy.md](./license-policy.md) for the license rules the diff
enforces.
