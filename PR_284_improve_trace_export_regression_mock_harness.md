# PR #284 — Improve trace export for regression and mock harness

## Summary

Resolves issue #284 by hardening the regression test harness, mock ledger
override API, and related CLI validation. Four concrete problems were fixed:

1. **`MergeLedgerOverrides` mutated the caller's map in-place** — overrides were
   written directly into `base`, so the caller's original map was silently
   modified after every merge. Subsequent calls sharing the same base saw
   stale keys from prior merges.

2. **Empty transaction hash produced a terse, non-actionable error** —
   `testTransaction` returned `"transaction hash is empty; skip this entry"`.
   This gave no guidance to the operator on what went wrong or how to fix it.

3. **`--workers` documentation/behavior mismatch** — the Long description said
   "must be a positive integer" but the validator only rejected `< 0`, allowing
   `0` which is silently auto-corrected to `4` in `runRegressionTest`. One test
   suite (the `cmd` test) would have failed on this description check after the
   Long text was updated.

4. **Missing test coverage** for `MergeLedgerOverrides` isolation, the improved
   empty-hash message, `--workers=0` pass-through behavior, and additional
   `LoadLedgerOverrideManifest` / `ParseLedgerOverrideFlags` edge cases.

---

## Changes

### `internal/simulator/mock.go`

**`MergeLedgerOverrides` — allocate a fresh map (no more mutation)**

Before:
```go
// Applied overrides directly into base — mutated the caller's map.
for key, value := range overrides {
    base[key] = value
}
return base
```

After:
```go
// Always allocates a new map; caller's base is never touched.
merged := make(map[string]string, len(base)+len(overrides))
for key, value := range base { merged[key] = value }
for key, value := range overrides { merged[key] = value }
return merged
```

The nil-base fast-path (`return base` when `len(overrides) == 0`) is preserved
for the zero-allocation case; only the mutating branch changed.

### `internal/simulator/regression_harness.go`

**`testTransaction` — actionable empty-hash message**

Old message:
```
transaction hash is empty; skip this entry
```

New message:
```
transaction hash is empty — cannot test an empty hash
  Fix: verify the transaction hash list from the RPC is not corrupted
  Tip: re-run with --verbose to see which transactions are being fetched
```

### `internal/cmd/regression_test.go`

- Updated the `Long` description's `--workers` line from  
  `"--workers must be a positive integer (defaults to 4)."` →  
  `"--workers must be >= 0; when 0 it defaults to 4."`  
  so the description accurately reflects the actual behavior.

### `docs/regression-test-command.md`

- Updated `--workers` flag table entry: "Must be a positive integer" →
  "When 0 or not set, defaults to 4. Must not be negative."
- Updated the Validation table with a new row: `--workers = 0` →
  "Silently auto-corrected to `4` (not an error)"
- Added a new **Mock Harness — Ledger Entry Overrides** section documenting
  `--mock-ledger-manifest` and `--mock-ledger-entry` flags, their validation
  rules, error message format, and merge behavior.

---

## New tests

### `internal/simulator/mock_test.go` — 7 new tests

| Test | What it verifies |
|---|---|
| `TestMergeLedgerOverrides_DoesNotMutateBase` | Caller's base map is unchanged after merge |
| `TestMergeLedgerOverrides_NewKeyInOverride` | New key from override appears in result, base untouched |
| `TestMergeLedgerOverrides_BothNilAndEmpty` | `nil + nil` → `nil`, no panic |
| `TestMergeLedgerOverrides_ResultIsIndependent` | Mutating the returned map does not affect base |
| `TestLoadLedgerOverrideManifest_KeysArePreserved` | Multiple entries loaded correctly |
| `TestLoadLedgerOverrideManifest_EmptyPath` | Empty path → error mentioning `--mock-ledger-manifest` |
| `TestParseLedgerOverrideFlags_ValueWithColon` | Value containing colon is parsed correctly via `SplitN` |

### `internal/simulator/regression_harness_test.go` — 4 new tests

| Test | What it verifies |
|---|---|
| `TestRegressionHarness_TestTransaction_EmptyHash_ActionableMessage` | Improved message contains `Fix:` hint and is not the old terse string |
| `TestRegressionTestSuite_StatisticsConsistency` | pass/fail/error counters match the actual result list |
| `TestRegressionTestSuite_FailedResults_ExcludesPassed` | `FailedResults()` never includes `"pass"` entries |
| `TestNewRegressionHarness_ZeroWorkers_DefaultsToFour` | `NewRegressionHarness(..., 0)` sets `MaxWorkers=4` |

### `internal/cmd/regression_cmd_test.go` — 6 new tests

| Test | What it verifies |
|---|---|
| `TestValidateRegressionFlags_ZeroWorkers_Passes` | `--workers=0` is not rejected by `PreRunE` |
| `TestValidateRegressionFlags_NegativeWorkers_IsRejected` | `--workers=-5` is caught with `--workers` in the message |
| `TestRegressionTestCmd_LongDescriptionMentionsWorkers` | Long text references `--workers` |
| `TestRegressionTestCmd_LongDescriptionAccurateForZeroWorkers` | Long text does not say "must be a positive integer" |
| `TestValidateRegressionFlags_CountAtExactMaximum` | `--count=1000` (boundary) is accepted |
| `TestValidateRegressionFlags_CountOneOverMaximum` | `--count=1001` is rejected and includes the count value |

---

## Test coverage summary

All pre-existing tests continue to pass. The new tests exercise:

- The no-mutation guarantee in `MergeLedgerOverrides` (the fixed bug)
- The improved error message in `testTransaction` for empty hashes
- The `--workers=0` auto-correction documented in `Long`
- Boundary conditions at the exact max count (1000) and one over (1001)
- Isolation: returned merge map is independent of the original base

---

## How to review

Key files:
1. `internal/simulator/mock.go` — `MergeLedgerOverrides` allocation fix
2. `internal/simulator/regression_harness.go` — empty-hash message
3. `internal/cmd/regression_test.go` — Long description accuracy
4. `docs/regression-test-command.md` — workers clarification + mock harness section

---

## Checklist

- [x] `MergeLedgerOverrides` no longer mutates the caller's base map
- [x] Empty transaction hash returns an actionable error with `Fix:` hint
- [x] `--workers` Long description matches actual code behavior (0 → auto-correct to 4)
- [x] Validation table in docs updated for `--workers=0` auto-correction
- [x] Mock harness flag documentation added to `regression-test-command.md`
- [x] 17 new tests covering all fixed behaviors and edge cases
- [x] No new dependencies introduced
- [x] PR description file added to `.gitignore`

---

## Related

Closes #284
