# Release-note fragments

This directory holds pending release-note fragments. Each fragment is a single
TOML file describing one user-facing change. The `make changelog-generate`
target assembles them into `CHANGELOG.md` and the `make changelog-check` target
validates them in CI.

See `docs/changelog-fragments.md` for the full authoring guide.

---

## Filename convention

Name every fragment file `<pr-number>-<slug>.toml`, where:

- `<pr-number>` is the GitHub PR number (integer, e.g. `597`).
- `<slug>` is a short lowercase hyphenated description (e.g. `api-compat-checks`).

Examples:

```
597-api-compat-checks.toml
1060-session-schema-v4-tags.toml
```

The validator enforces uniqueness of the `pr` field inside each file, not the
filename itself, but keeping the number in the filename makes the directory
easy to scan and avoids accidental reuse.

The file `example-cli-flag.toml` is a copy-able template. It is skipped by
both the validator and the generation script; do not use it as a real fragment.

---

## Validating fragments locally

Run the validator against all pending fragments before pushing:

```bash
make changelog-check
```

To validate a single file:

```bash
bash scripts/validate-fragments.sh changelog/fragments/597-api-compat-checks.toml
```

A passing run prints one `[OK]` line per fragment and exits 0:

```
[OK]  597-api-compat-checks.toml
[OK]  All fragments valid.
```

---

## Duplicate PR number detection

Each fragment's `pr` field must be unique across all pending fragments.
Reusing the same PR number produces an error and exits 1:

```
[FAIL] 598-another-change.toml: duplicate pr=597 already used in 597-api-compat-checks.toml

1 error(s) found. Fix fragments before releasing.
```

To resolve it, correct the `pr` field to match the actual GitHub PR number.

---

## Breaking change fragments

Set `breaking = true` when users must change their invocation, scripts, or
integration code after the release. The validator enforces a non-empty
`migration_note` whenever `breaking = true`:

```toml
category = "breaking"
pr = 900
summary = "Remove --legacy-output flag"
breaking = true
migration_note = "Replace --legacy-output with --json. See MIGRATION_GUIDE.md."
affects = ["cli-flags"]
```

A missing or empty `migration_note` with `breaking = true` fails validation:

```
[FAIL] 900-remove-legacy-output.toml: breaking=true requires a non-empty migration_note

1 error(s) found. Fix fragments before releasing.
```

Breaking fragments are placed at the top of the generated release section
(before all other categories) and trigger a highlighted warning block pointing
to `MIGRATION_GUIDE.md`. Every breaking fragment also requires:

- An entry in `docs/compatibility-matrix.md` updating the affected surface's
  stability level.
- A before/after example in `MIGRATION_GUIDE.md`.
- Updated API snapshot files (CI snapshot check fails until they are
  regenerated).
