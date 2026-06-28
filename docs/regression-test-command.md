# `glassbox regression-test` — Regression Test Command Reference

The `regression-test` command fetches historic failed transactions from the Stellar network
and verifies that the local simulator produces identical results, ensuring protocol changes
do not introduce regressions.

---

## Synopsis

```sh
glassbox regression-test [flags]
```

---

## Flags

| Flag | Default | Description |
|---|---|---|
| `--count` | `100` | Number of historic failed transactions to test. Must be between 1 and 1000 (inclusive). |
| `--workers` | `4` | Number of parallel test workers. When 0 or not set, defaults to 4. Must not be negative. |
| `--network`, `-n` | `mainnet` | Stellar network: `testnet`, `mainnet`, or `futurenet`. |
| `--rpc-url` | _(config)_ | Custom RPC URL. Overrides the default for the selected network. |
| `--rpc-token` | _(env: `GLASSBOX_RPC_TOKEN`)_ | RPC authentication token. |
| `--start-seq` | `0` | Starting ledger sequence number (`0` = most recent). |
| `--protocol-version` | `0` | Override protocol version for all tests (`0` = use default). |
| `--verbose`, `-v` | `false` | Show per-transaction progress output. |

---

## Validation

All flags are validated in `PreRunE` before any network or simulator calls are made:

| Condition | Error message |
|---|---|
| `--count` ≤ 0 | `--count must be greater than 0 (got N)` |
| `--count` > 1000 | `--count N exceeds the maximum of 1000` |
| `--workers` < 0 | `--workers must be a positive integer (got N)` |
| `--workers` = 0 | Silently auto-corrected to `4` (not an error) |
| `--network` unknown | `invalid --network "…"; must be one of: testnet, mainnet, futurenet` |
| `--protocol-version` unsupported | `invalid --protocol-version N: …` with a hint to run `glassbox version` |

Failures are surfaced immediately, before consuming any network quota.

---

## Output

On completion, a summary is printed:

```
Regression Test Summary:
  Total Tests: 100
  Passed:      97
  Failed:      2
  Errors:      1
  Success Rate: 97.0%
```

When failures occur, the first 10 are listed with their transaction hash and error message.
A hint is included to run `glassbox debug <tx-hash>` for a detailed trace of any failing
transaction.

---

## Exit Codes

| Code | Meaning |
|---|---|
| `0` | All tests passed |
| `1` | One or more tests failed, or a validation error occurred |
| `2` | Simulator binary not found — run `glassbox doctor --fix` |

---

## Examples

```sh
# Run 100 regression tests on mainnet (default)
glassbox regression-test --count 100

# Use more parallel workers for faster runs
glassbox regression-test --count 1000 --workers 8

# Test against a specific protocol version
glassbox regression-test --count 500 --network mainnet --protocol-version 22

# Verbose output shows per-transaction progress
glassbox regression-test --count 50 --verbose

# Use a custom RPC endpoint
glassbox regression-test --count 200 --rpc-url https://my-rpc.example.com
```

---

## Mock Harness — Ledger Entry Overrides

The regression harness and `glassbox debug` both support injecting synthetic ledger entries to test
contract behavior against controlled state without hitting the network.

### `--mock-ledger-manifest <file>`

Load a JSON manifest file mapping base64-encoded XDR keys to base64-encoded XDR entry values:

```json
{
  "ledger_entries": {
    "<base64-xdr-key>": "<base64-xdr-entry>"
  }
}
```

**Validation:**

| Condition | Error message |
|---|---|
| File not found | `--mock-ledger-manifest: file not found: "…"` with path hint |
| Invalid JSON | `--mock-ledger-manifest: failed to parse "…" as JSON` with format hint |
| Empty entry value | `--mock-ledger-manifest: entry "…" has an empty value` |
| Invalid base64 value | `--mock-ledger-manifest: entry "…" has an invalid base64 value` |

### `--mock-ledger-entry key:value`

Inject a single ledger entry inline. Each value must be `key:value` where both are non-empty
and the value is valid base64-encoded XDR.

**Validation:**

| Condition | Error message |
|---|---|
| Missing `:` separator | `--mock-ledger-entry: invalid format "…" — expected key:value` |
| Empty key | `--mock-ledger-entry: invalid format "…"` (key must not be empty) |
| Empty value | `--mock-ledger-entry: entry "…" has an empty value` |
| Invalid base64 | `--mock-ledger-entry: entry "…" has an invalid base64 value` |

### Merge behavior

When both `--mock-ledger-manifest` and `--mock-ledger-entry` are used together, entries from
`--mock-ledger-entry` take precedence over keys in the manifest (last-writer wins). The merge
never mutates the original manifest map — it always returns a fresh copy.

---

## See Also

- [`glassbox debug`](./debug-command.md) — debug a single transaction in detail
- [`glassbox doctor`](./sandboxed-replay.md) — environment setup checker
- [`glassbox version`](./debug-command.md) — check supported protocol versions
