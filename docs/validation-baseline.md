# Validation Baseline

This document records the commands, expected outcomes, and pre-existing
constraints for the focused changes made in this session. Any contributor can
reproduce this procedure from a clean checkout.

## Pre-existing constraint: Go not installed in this environment

The machine's PATH lists `C:\Program Files\Go\bin` but the directory does not
exist on disk — Go is not installed. All `go test` commands below are therefore
documented as **expected commands** with their expected outcomes; they cannot
be run from this shell environment.

This constraint is pre-existing and not introduced by this change set.
To verify on a machine with Go installed:

```sh
# From the repository root
go version  # expected: go1.22.x or later
```

---

## Area 1 — Provider isolation (`internal/signer`)

### Command

```sh
go test ./internal/signer/... -v -run .
```

### Expected outcome

All existing tests pass. No new test files were added for this area; the fix
was documentation only (ADR-006, README.md).

### Pre-existing failures

None known. The signer tests do not require a real PKCS#11 token or AWS
credentials; the KMS and PKCS#11 tests are stubbed/mocked.

---

## Area 2 — Data classification and audit retention (`internal/audit`)

### Command

```sh
go test ./internal/audit/... -v -run .
```

### Expected new tests to pass

| Test | File | What it verifies |
|------|------|-----------------|
| `TestRetention_MultiConstraintAdditive` | `rotation_test.go` | MaxAge + MaxSizeBytes are additive; a segment removed by either constraint is in the Remove list |
| `TestRetention_ZeroConfigKeepsAll` | `rotation_test.go` | Zero `RetentionConfig` removes nothing |
| `TestRetention_ActiveSegmentNotEligible` | `rotation_test.go` | `current.jsonl` never appears in Keep or Remove |

### Pre-existing failures

None known. All three new tests use only the existing internal `planRetention`
function and `OpenWriter`/`PlanRetention` APIs; no new production code was
added.

---

## Area 3 — Offline guarantees (`internal/offline`)

### Command

```sh
go test ./internal/offline/... -v -run .
```

### Expected new tests to pass

| Test | File | What it verifies |
|------|------|-----------------|
| `TestAirGapped_ValidateChecksBothLoadSites` | `offline_test.go` | `LoadEnvelopeFile` rejects tampered XDR at both stage-2 and stage-3 load sites |
| `TestAirGapped_SubmitDoesNotVerifySignatures` | `offline_test.go` | `SubmitSignedEnvelope` input validation hits the URL check, not a signature check — confirming it does not call `VerifySignatures` |

### Pre-existing failures

None known. The new tests do not make real network calls; `SubmitSignedEnvelope`
is exercised with an empty URL which triggers early input validation.

---

## Area 4 — Narrow terminal help (`internal/terminal`, `internal/clioutput`)

### Commands

```sh
go test ./internal/terminal/... -v -run .
go test ./internal/clioutput/... -v -run .
```

### Expected new tests to pass — `internal/terminal`

| Test | File | What it verifies |
|------|------|-----------------|
| `TestANSIRenderer_NOColorOverridesFORCECOLOR` | `ansi_test.go` | `IsTTY()` returns false and `Colorize` returns plain text when both `NO_COLOR` and `FORCE_COLOR` are set |
| `TestANSIRenderer_Symbol_CrossDivergence/TTY_mode_returns_[FAIL]` | `ansi_test.go` | `Symbol("cross")` returns `"[FAIL]"` in TTY mode |
| `TestANSIRenderer_Symbol_CrossDivergence/non-TTY_mode_returns_[X]` | `ansi_test.go` | `Symbol("cross")` returns `"[X]"` in non-TTY mode |
| `TestANSIRenderer_Symbol_ConsistentForNonCross/*` | `ansi_test.go` | check / warn / arrow_r / arrow_l / pin return the same string in both TTY and non-TTY modes |

### Expected new tests to pass — `internal/clioutput`

| Test | File | What it verifies |
|------|------|-----------------|
| `TestWantsJSON_CaseInsensitive` | `output_test.go` | `"JSON"`, `"Json"`, `"jSoN"` all return true; `"TEXT"` returns false |
| `TestWantsJSON_WhitespaceTrimmed` | `output_test.go` | `"  json  "`, `"  JSON  "` all return true |
| `TestWantsJSON_FlagTakesPrecedence` | `output_test.go` | `jsonFlag=true` returns true regardless of format string |

### Pre-existing failures

None known. All new tests exercise only pure functions (`WantsJSON`,
`ANSIRenderer`) with no external dependencies.

---

## Documentation-only changes (no test command)

The following files were updated with documentation fixes only. No automated
validator exists for markdown correctness beyond link checking; the content was
verified by cross-referencing each claim against the source files listed in the
Implementation tables.

| File | Key corrections |
|------|----------------|
| `docs/adr/006-provider-isolation.md` | Signer interface: 3→4 methods (added `KeyOrigin`); mock provider path via registry documented; concrete shell examples added |
| `docs/adr/004-data-classification.md` | Boundary E: SegmentManifest structure, chain-hash, RetentionConfig, and retention example added; Implementation table updated |
| `docs/adr/007-offline-guarantees.md` | §3: EnvelopeFile structure corrected (XDR not canonical JSON, checksum of XDR not payload); submission stage no-reverify note added; RPC failover table updated to match config file |
| `docs/adr/README.md` | Broken links to non-existent ADR-001 and ADR-002 removed; Related Documentation section corrected |
| `docs/no-color.md` | Removed false `GLASSBOX_NO_COLOR` env-var claim; corrected 5-step precedence order to match `checkTTY()`; added `TERM=dumb` to table; documented `NO_COLOR` overrides `FORCE_COLOR`; added Symbol cross divergence section |
| `docs/json-output.md` | Added error envelope section with JSON example, field table, stable error codes, and jq automation example |

---

## Markdown link check (manual)

```sh
# Check for broken internal links in docs/adr/
grep -r '\[.*\](.*\.md)' docs/adr/ | grep -v 'http' | awk -F'(' '{print $2}' | tr -d ')' \
  | while read f; do
      dir=$(dirname "$f")
      [ -f "docs/adr/$f" ] || echo "BROKEN: $f"
    done
```

Expected: no output (no broken links). The previously broken `001-snapshot-caching-strategy.md`
and `002-hsm-integration.md` references have been removed from `docs/adr/README.md`.
