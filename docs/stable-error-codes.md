# Stable Error Code Policy

Glassbox commands surface failures through a shared error type so that
automation scripts and CI pipelines can key on a **stable code** rather than
parsing free-form text.

## Error type

Every structured error is an `*errors.ErstError` (defined in
`internal/errors/errors.go`):

```go
type ErstError struct {
    Code    ErstErrorCode  // stable string, e.g. "RPC_CONNECTION_FAILED"
    Message string         // human-readable; may change between releases
    OrigErr error          // original cause, preserved via Unwrap()
    Hint    string         // optional actionable remediation for the user
}
```

- `Code` is the **automation contract** — do not key on `Message`.
- `Hint` is never included in `Error()` so it does not pollute log entries.
  It is surfaced by `errors.Hint(err)` and printed separately.
- `OrigErr` is returned by `Unwrap()` so `errors.Is` / `errors.As` chains
  work transparently.

## Output adapters (internal/clioutput)

| Function | When to use |
|---|---|
| `clioutput.WriteError(w, cmd, err, ctx)` | JSON mode — writes a schema-versioned envelope with `error.code` |
| `clioutput.FormatErrorText(err)` | Text mode — returns `[CODE] CODE: message\nHint: …` |

Both adapters extract the same `Code` from the error, so text and JSON modes
are always consistent. Because `ErstError.Error()` already starts with
`CODE: `, the code appears twice in `FormatErrorText` output and once at the
start of the JSON `message` field.

### What the CLI process prints today

No command currently routes its failure through `WriteError`; the adapters are
covered by `internal/clioutput/error_test.go`. When a command returns an error,
`cmd/glassbox/main.go` writes to **stderr**:

```text
Error: <CODE>: <message>[: <cause>]
Hint: <remediation>          # only when the error carries a Hint
```

and exits with `ExitCodeFor(err)`. Errors that are not `ErstError` print
`Error: <message>` with no code and exit `3`. See
[json-output-automation.md](json-output-automation.md) for filtering on
`error.code` from scripts.

`ErstError.Error()` already starts with the code (`CODE: message`), so
`FormatErrorText` output shows the code twice, for example
`[RPC_CONNECTION_FAILED] RPC_CONNECTION_FAILED: RPC connection failed: …`.

### Default stderr output

When a command returns an error, `cmd/glassbox/main.go` prints it to stderr and
adds the hint on its own line when one is set:

```
Error: TRANSACTION_NOT_FOUND: transaction not found: <underlying cause>
Hint: Check the transaction hash and confirm --network matches where it was submitted (testnet, mainnet, or futurenet). If the transaction is recent the RPC may not have indexed it yet — retry shortly or use --watch to wait for it.
```

If the error has no hint, the `Hint:` line is omitted. The JSON adapter
(`WriteError`) puts the same text in `error.remediation` and omits the field
when it is empty.

### JSON envelope shape

```jsonc
{
  "schema_version": "1.0",
  "glassbox_version": "1.2.3",
  "generated_at": "2026-07-24T12:00:00Z",
  "command": "debug",
  "data": {
    "error": {
      "code": "RPC_CONNECTION_FAILED",   // ← stable, key on this
      "severity": "error",
      "message": "RPC_CONNECTION_FAILED: RPC connection failed: dial tcp …",
      "remediation": "The RPC endpoint could not be reached. …",
      "context": {
        "network": "testnet"
      }
    }
  }
}
```

### Error payload fields

Defined by `ErrorPayload` in `internal/clioutput/error.go`:

| Field | Type | Always present | Notes |
|---|---|---|---|
| `code` | string | yes | Stable `ErstErrorCode`; `UNKNOWN` when the error is not an `ErstError` |
| `severity` | string | yes | Currently always `error` |
| `message` | string | yes | Full `err.Error()` text including wrapped causes; may change between releases |
| `remediation` | string | no (`omitempty`) | The error's `Hint`; static text written in `internal/errors` or the command |
| `context` | object of string → string | no (`omitempty`) | Command-supplied `ctx` map passed to `WriteError`; omitted when `nil` or empty |

## Context fields and log aggregation

`context` is a flat `map[string]string`. Glassbox does not add keys on its
own: every key comes from the `ctx` argument of `clioutput.WriteError`. The
only keys exercised today are `tx_hash` and `network`
(`TestWriteError_ContextIncluded` in `internal/clioutput/error_test.go`).

Use this classification when indexing error envelopes in a log aggregator,
and when choosing keys for new `WriteError` call sites:

| Field / key | Safe to index | Why |
|---|---|---|
| `code` | yes | Fixed vocabulary from the [catalogue](#stable-code-catalogue); low cardinality |
| `severity` | yes | Fixed vocabulary |
| `command` (envelope) | yes | Cobra command name |
| `schema_version`, `glassbox_version` | yes | Build metadata |
| `context.network` | yes | One of `testnet`, `mainnet`, `futurenet` or a configured network name |
| `context.tx_hash` | yes | Public on-chain identifier; high cardinality, so index as a keyword, not a label |
| `remediation` | store, don't index | Static prose; can change between releases |
| `message` | store only, redact first | Contains the wrapped cause verbatim, which can include RPC URLs (query-string tokens), local file paths, and user input |

Rules for new context keys:

- Use `snake_case` keys and keep values to identifiers (network, hash,
  ledger sequence, command name).
- Never put secrets, tokens, private keys, full RPC URLs, or absolute home
  paths in `context`. The same redaction expectations apply as in
  `glassbox doctor --bundle` (see [diagnostics-bundle.md](diagnostics-bundle.md))
  and execution plans (`internal/plan`, which redacts URL tokens and keys).
- Add the key and its classification to the table above in the same change.

Verify the payload contract with:

```bash
go test ./internal/clioutput/ -run 'WriteError|FormatErrorText'
```

## Stable code catalogue

### General

| Code | Meaning | Exit |
|---|---|---|
| `VALIDATION_FAILED` | Bad flag, argument, or config value | 1 |
| `ARGUMENT_REQUIRED` | Reserved; not currently emitted. Missing flags (`WrapCliArgumentRequired`) report `VALIDATION_FAILED` | 1 |
| `CONFIG_ERROR` | Config file unreadable or invalid | 2 |
| `UNKNOWN` | Unclassified error | 3 |

### RPC

| Code | Meaning | Exit |
|---|---|---|
| `RPC_CONNECTION_FAILED` | Endpoint unreachable | 3 |
| `RPC_TIMEOUT` | Request timed out | 3 |
| `ALL_RPC_FAILED` | All failover endpoints failed | 3 |
| `RPC_ERROR` | RPC server returned a non-200 / error response | 3 |
| `TRANSACTION_NOT_FOUND` | TX hash not found on the selected network | 1 |
| `LEDGER_NOT_FOUND` | Ledger sequence unavailable | 1 |
| `LEDGER_ARCHIVED` | Ledger has been archived | 1 |
| `RATE_LIMIT_EXCEEDED` | RPC rate limit hit | 1 |
| `UNAUTHORIZED` | Missing or invalid auth token | 1 |
| `INVALID_NETWORK` | Unrecognised `--network` value | 1 |
| `NETWORK_NOT_FOUND` | Named network has no known endpoint | 1 |

### Simulator

| Code | Meaning | Exit |
|---|---|---|
| `SIMULATOR_NOT_FOUND` | Simulator binary absent or not executable | 2 |
| `SIMULATION_FAILED` | Simulator process returned non-zero | 3 |
| `SIMULATOR_CRASH` | Simulator process crashed unexpectedly | 3 |
| `SIMULATION_LOGIC_ERROR` | Contract logic error detected | 1 |

### Other

| Code | Meaning | Exit |
|---|---|---|
| `SOURCE_DISCOVERY_FAILED` | Source mapping unavailable for contract | 3 |
| `ANALYSIS_TRUNCATED` | Analyzer stopped early because a resource budget ran out | 3 |
| `INTERNAL_ERROR` | Unexpected internal failure (`WrapInternal`) | 3 |

### Session concurrency (#813)

| Code | Meaning | Exit |
|---|---|---|
| `SESSION_WRITE_CONFLICT` | A concurrent writer saved a newer revision; reload or use `--force` | 1 |
| `SESSION_LOCK_HELD` | Advisory lock held by a live process; retry after it exits | 1 |

### Audit signing and verification

| Code | Meaning | Exit |
|---|---|---|
| `KMS_THROTTLED` | AWS KMS throttled requests and the retry budget ran out | 3 |
| `KMS_UNAUTHORIZED` | AWS KMS rejected Sign/GetPublicKey (IAM, disabled key, pending deletion) | 3 |
| `KMS_TRANSIENT_FAILURE` | AWS KMS returned transient errors and the retry budget ran out | 3 |
| `AUDIT_DIR_POLICY_VIOLATION` | `audit:verify-dir` found policy violations | 3 |

### Reserved (defined, not currently emitted)

These constants exist in `internal/errors/glassbox_error_code.go`, but no
wrapper returns them yet. Do not key automation on them until a command emits
them.

| Code | Notes |
|---|---|
| `RPC_INVALID_RESPONSE` | `WrapRPCInvalidResponse` returns `*RPCInvalidResponseError`, not an `ErstError`, so JSON output reports `UNKNOWN` (exit 3). |
| `FOOTPRINT_EMPTY`, `FOOTPRINT_OVERLAP`, `FOOTPRINT_OVERSIZED`, `FOOTPRINT_UNKNOWN_KEY_TYPE`, `FOOTPRINT_XDR_VERSION` | Not referenced outside the constant block. |

---

## Remediation text by code

This table lists the remediation hint that each `Wrap*` constructor in
`internal/errors/errors.go` attaches by default. That hint is what appears on
the `Hint:` line (and in `error.remediation` from the JSON adapter). A call site can attach a different
hint with `(*ErstError).WithHint`, so a specific command may print more detailed
guidance.

When a code has **no built-in hint**, no `Hint:` line is printed. The
"Recommended action" column then gives the documented remediation, which comes
from the [operator runbook](./operator-runbook.md), not from the binary.

| Code | Built-in hint (verbatim, from the wrapper) | Recommended action when no hint is printed |
|---|---|---|
| `TRANSACTION_NOT_FOUND` | "Check the transaction hash and confirm --network matches where it was submitted (testnet, mainnet, or futurenet). If the transaction is recent the RPC may not have indexed it yet — retry shortly or use --watch to wait for it." | — |
| `RPC_CONNECTION_FAILED` | "The RPC endpoint could not be reached. Check your internet connection and the endpoint, pass a known-good one with --rpc-url <url>, and make sure it serves the selected --network." | — |
| `RPC_TIMEOUT` | "The RPC endpoint did not respond in time. It may be overloaded or slow — retry, raise the timeout, or switch to a different endpoint with --rpc-url <url>." | — |
| `ALL_RPC_FAILED` | `WrapAllRPCFailed`: "All configured RPC endpoints failed. Provide a working endpoint with --rpc-url <url>, run 'glassbox doctor' for a full environment check, or check 'glassbox status --network <name>'." `WrapAllRPCFailedWithDiagnostics` uses the per-endpoint failure summary as the hint. | — |
| `SOURCE_DISCOVERY_FAILED` | Caller-supplied, or by default: "Provide the contract source directory with --contract-source <path>, or recompile with 'debug = true' in [profile.release] for DWARF symbols. Run 'glassbox doctor' for a full environment check." | — |
| `SESSION_WRITE_CONFLICT` | "Another Glassbox process saved this session while you were editing it. Run 'glassbox session resume <id>' to reload the latest version, or re-run your save with --force to overwrite it." | — |
| `SESSION_LOCK_HELD` | "Another Glassbox instance is currently saving this session. Wait for it to finish and retry. If the other process has crashed, the lock will be cleared automatically after 5 minutes." | — |
| `ANALYSIS_TRUNCATED` | "The analyzer reached a configured resource limit before completing. … Raise --analyzer-timeout, --max-nodes, --max-depth, or --max-input-bytes to analyze the full trace." | — |
| `KMS_THROTTLED` | "AWS KMS is throttling requests. Increase GLASSBOX_KMS_MAX_RETRIES / GLASSBOX_KMS_MAX_BACKOFF_MS to absorb burst traffic, or reduce the signing rate. Use --audit-log-kms-key-id to confirm the correct key." | — |
| `KMS_UNAUTHORIZED` | "Verify the IAM policy attached to this identity includes kms:Sign and kms:GetPublicKey for the target key, and that the key is Enabled (not Disabled or PendingDeletion). See docs/audit-kms-signing.md for the minimum IAM policy." | — |
| `KMS_TRANSIENT_FAILURE` | "AWS KMS returned a transient error. This is usually a brief AWS service disruption. Retry the command or increase GLASSBOX_KMS_MAX_RETRIES. …" | — |
| `AUDIT_DIR_POLICY_VIOLATION` | "Run 'glassbox audit:verify-dir --json --dir <path>' for machine-readable details with per-file and aggregate results. Use --policy-config to load a policy file, or --expected-signers / --expected-schema-version to tighten checks." | — |
| `VALIDATION_FAILED` | Only from `WrapProtocolUnsupported`: "This build of Glassbox does not support that Soroban protocol version. …" Other validation wrappers attach none. | Read the message; it names the flag or value that is wrong. Run the command with `--help` to see accepted values. |
| `UNAUTHORIZED` | *(none)* | Check that `GLASSBOX_RPC_TOKEN` or `rpc_token` is set and not expired. See [Authentication errors](./operator-runbook.md#authentication-errors) and [RPC bearer token examples](./rpc-token-examples.md). |
| `RATE_LIMIT_EXCEEDED` | *(none)* | Wait and retry, lower request volume, or switch endpoint with `--rpc-url`. See [Rate limiting](./operator-runbook.md#rate-limiting). |
| `INVALID_NETWORK` | *(none; the message lists the valid values)* | Use `testnet`, `mainnet`, `futurenet`, or a custom network defined in config. |
| `NETWORK_NOT_FOUND` | *(none)* | Define the network's endpoint in config, or pass `--rpc-url`. |
| `LEDGER_NOT_FOUND` / `LEDGER_ARCHIVED` | *(none)* | Use an RPC or archive provider that retains that ledger range. |
| `CONFIG_ERROR` | *(none)* | Run `glassbox config show --explain` to find the bad value and its source file. |
| `SIMULATOR_NOT_FOUND` | *(none)* | Build or install the simulator, or set `GLASSBOX_SIMULATOR_PATH`. Run `glassbox doctor`. |
| `SIMULATION_FAILED` / `SIMULATOR_CRASH` | *(none; simulator stderr is included in the message)* | Re-run with `--verbose` and attach a diagnostics bundle (`glassbox doctor --bundle`). |
| `RPC_ERROR` | *(none; message includes endpoint URL and server code)* | Check the server code in the message. For 4xx, check the token and method; for 5xx, retry or switch endpoint. |
| `INTERNAL_ERROR` / `UNKNOWN` | *(none)* | Generate a diagnostics bundle and open an issue. |

### Example: failure with remediation

```sh
glassbox debug --network testnet <unknown-tx-hash>
echo $?
```

Expected stderr and exit code:

```
Error: TRANSACTION_NOT_FOUND: transaction not found: <underlying cause>
Hint: Check the transaction hash and confirm --network matches where it was submitted (testnet, mainnet, or futurenet). If the transaction is recent the RPC may not have indexed it yet — retry shortly or use --watch to wait for it.
1
```

### Example: failure without remediation

`UNAUTHORIZED` has no built-in hint, so no `Hint:` line is printed:

```
Error: UNAUTHORIZED: unauthorized
1
```

Automation should match the code prefix on the `Error:` line, then look up the
recommended action in the table above.

> **Note:** No command currently calls `clioutput.WriteError` or
> `clioutput.FormatErrorText`. Errors reach users only through the stderr
> format above, including when `--json` is passed. The
> [JSON envelope shape](#json-envelope-shape) describes the adapter contract
> for commands that adopt it. You can confirm this with
> `grep -rn 'clioutput.WriteError' --include=*.go internal cmd`, which prints
> nothing outside `internal/clioutput`.

### Verifying the table

From the repository root:

```sh
# Every hint the wrappers attach
grep -n 'Hint:' internal/errors/errors.go
```

Each `Hint:` line should match one row in the "Built-in hint" column above.

---

## Cross-reference: error codes and SARIF rules

The same failure can surface in two forms: as a CLI error code in `--json`
output and as a SARIF rule ID in `--format sarif` output. These are
**related but not identical**: an error code describes why a command failed,
while a SARIF rule ID describes a specific class of finding inside a run.

No runtime mapping is performed between the two systems — the relationship is
informational only, to help you correlate outputs without mistaking one for
the other.

| ErstErrorCode | Related SARIF rule(s) | Nature of relationship |
|---|---|---|
| `SOURCE_DISCOVERY_FAILED` | `GB1002` (`SourceMapDwarfMissing`) | Both indicate a contract without usable debug symbols, but `SOURCE_DISCOVERY_FAILED` means the whole source-mapping stage was skipped; `GB1002` is emitted per-run when the binary has no DWARF sections at all. One does not imply the other. |
| `SOURCE_DISCOVERY_FAILED` | `GB1001` (`SourceMapOffsetNotFound`) | `GB1001` is a narrower condition — a single WASM offset failed to resolve — and does not cause command failure. `SOURCE_DISCOVERY_FAILED` is fatal to the source-mapping pipeline. |
| `SOURCE_DISCOVERY_FAILED` | `GB1003` (`SourceMapNoGitRepository`) | `GB1003` is informational (`note` level) and does not fail the run; `SOURCE_DISCOVERY_FAILED` does. No direct causal link. |
| `SIMULATION_LOGIC_ERROR` | `GB2001` (`DeprecatedHostFunction`) | A deprecated host function call is a SARIF warning, not a simulation logic error. `SIMULATION_LOGIC_ERROR` means the contract itself failed; `GB2001` means a specific function is deprecated and may fail in a future protocol release. |
| `SIMULATION_LOGIC_ERROR` | `GB2002` (`AllowlistedUnsafeHostFunction`) | Similarly, `GB2002` is a security finding about an unsafe-but-permitted call; it does not cause `SIMULATION_LOGIC_ERROR` unless the unsafe call is also the cause of the simulation trap. |
| `VALIDATION_FAILED` | *(no direct mapping)* | Validation failures happen before analysis begins; no SARIF output is produced. |
| `RPC_CONNECTION_FAILED` | *(no direct mapping)* | Network errors prevent the run from producing any SARIF findings. |

### Reading correlations across output formats

When a `glassbox scan` or `glassbox debug` run produces both `--json` and
`--format sarif` output, use the following approach to correlate them:

1. Check `--json` for `error.code` to understand why the command failed.
2. Inspect the SARIF `results[]` array for rule IDs in the `GB1xxx`/`GB2xxx`
   range — these are diagnostic findings from the same run.
3. Do not assume a SARIF result means the corresponding error code was set, or
   vice versa.  A run can produce `GB1001` warnings and still exit 0; and a
   `SOURCE_DISCOVERY_FAILED` error can occur without any `GB1xxx` result if
   the discovery failure happened before SARIF export ran.

For the full SARIF rule catalogue see
[sarif-diagnostic-rules.md](sarif-diagnostic-rules.md) and
[sarif-export.md](sarif-export.md).

## Adding a new code

1. Add the constant to `internal/errors/glassbox_error_code.go`.
2. Add it to the `codeToSentinel` map with a matching sentinel error.
3. Add it to `errorCodeRegistry` in `errors.go` if a sentinel should map to it.
4. Update the table above.
5. Decide whether it maps to exit code 1, 2, or 3 and add it to
   `userErrorCodes` or `configErrorCodes` in `internal/cmd/exitcode.go`.

## Exit code taxonomy

| Code | Meaning |
|---|---|
| 0 | Success |
| 1 | User error (bad input, validation failure) |
| 2 | Config / environment error |
| 3 | Internal / runtime error, and any code not listed in `userErrorCodes` or `configErrorCodes` |
| 130 | Interrupted (SIGINT or SIGTERM) or context cancelled |

This is the Go binary's taxonomy (`internal/cmd/exitcode.go`). The TypeScript
wrapper uses a different registry; see [Exit codes](./exit-codes.md).

## Migrated commands

The following commands emit `ErstError` for all failure paths and have test
coverage of at least one validation error and one runtime error:

- `debug`
- `trace`
- `audit:sign`
- `audit:verify`
- `session` (save, resume, list, delete, recover, doctor)
- `protocol:register`, `protocol:diagnose`, `protocol:verify`
- `config show` (load and explain paths)
