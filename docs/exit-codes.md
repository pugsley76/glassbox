# Glassbox Exit-Code Registry

This document is the authoritative cross-language reference for all exit codes
produced by the Glassbox CLI (TypeScript wrapper) and Go binary.  Shell scripts
and CI pipelines **must** key on these codes, never on message text.

> **The two registries are not the same.** The TypeScript wrapper
> (`src/exit-codes.ts`) uses codes 0–6, 130, and 143. The Go binary
> (`internal/cmd/exitcode.go`) uses only 0, 1, 2, 3, and 130. The same numeric
> value can mean different things: for example, `2` is `VALIDATION_ERROR` in
> TypeScript but a configuration error in Go. Check which binary your script
> invokes before you key on a code.

## Stable codes (TypeScript wrapper)

| Code | Name               | When used |
|-----:|--------------------|-----------|
|    0 | SUCCESS            | Command completed successfully. |
|    1 | UNKNOWN_ERROR      | Unexpected or unclassified runtime error. |
|    2 | VALIDATION_ERROR   | Invalid user input: bad arguments, malformed payload, wrong format. |
|    3 | CONFIGURATION_ERROR| Configuration or provider error: missing key, unsupported platform, registration failure. |
|    4 | NETWORK_ERROR      | Network or RPC error: connection refused, timeout, endpoint unavailable. |
|    5 | SECURITY_ERROR     | Security or authorization error: invalid signature, unauthorized origin, rate limit exceeded. |
|    6 | RESOURCE_ERROR     | Resource or concurrency error: lock acquisition failed, resource busy. |
|  130 | SIGINT             | Process received SIGINT (Ctrl-C). POSIX: 128 + 2. |
|  143 | SIGTERM            | Process received SIGTERM. POSIX: 128 + 15. |

Codes 128–143 are reserved for fatal signal termination per POSIX (128 + signal
number).  No application-level code may be assigned in that range.

## TypeScript registry

All TypeScript command wrappers import exit codes from `src/exit-codes.ts`:

```typescript
import { ExitCode } from '../exit-codes';
process.exit(ExitCode.VALIDATION_ERROR); // 2
```

The registry is an `as const` object so callers get full type inference:

```typescript
export const ExitCode = {
    SUCCESS:              0,
    UNKNOWN_ERROR:        1,
    VALIDATION_ERROR:     2,
    CONFIGURATION_ERROR:  3,
    NETWORK_ERROR:        4,
    SECURITY_ERROR:       5,
    RESOURCE_ERROR:       6,
    SIGINT:             130,
    SIGTERM:            143,
} as const;
```

## Go binary exit codes

The Go binary surfaces structured errors via `internal/errors/errors.go`.
Each `ErstError` carries a stable `Code` string (e.g. `"RPC_CONNECTION_FAILED"`).
`cmd/glassbox/main.go` passes the returned error to `cmd.ExitCodeFor`
(`internal/cmd/exitcode.go`), which picks the exit code. See
[Stable error codes](./stable-error-codes.md) for the full code catalogue,
including per-code remediation text.

| Exit | Go constant | ErstError codes |
|-----:|---|---|
| 0 | `ExitSuccess` | *(no error)* |
| 1 | `ExitUserError` | `VALIDATION_FAILED`, `ARGUMENT_REQUIRED`, `SIMULATION_LOGIC_ERROR`, `TRANSACTION_NOT_FOUND`, `LEDGER_NOT_FOUND`, `LEDGER_ARCHIVED`, `RATE_LIMIT_EXCEEDED`, `UNAUTHORIZED`, `INVALID_NETWORK`, `NETWORK_NOT_FOUND`, `SESSION_WRITE_CONFLICT`, `SESSION_LOCK_HELD` (the `userErrorCodes` map) |
| 2 | `ExitConfigError` | `CONFIG_ERROR`, `SIMULATOR_NOT_FOUND` (the `configErrorCodes` map) |
| 3 | `ExitInternalError` | Every other code, including `UNKNOWN`, `INTERNAL_ERROR`, `RPC_CONNECTION_FAILED`, `RPC_TIMEOUT`, `ALL_RPC_FAILED`, `RPC_ERROR`, `SIMULATION_FAILED`, `SIMULATOR_CRASH`, `SOURCE_DISCOVERY_FAILED`, `ANALYSIS_TRUNCATED`, `KMS_*`, `AUDIT_DIR_POLICY_VIOLATION` |
| 130 | `InterruptExitCode` | SIGINT **or SIGTERM** (`ErrInterrupted`), or `context.Canceled` |

How errors that are not `ErstError` values are mapped:

- They are first matched against sentinels with `errors.Is`.
  `ErrValidationFailed`, `ErrArgumentRequired`, `ErrTransactionNotFound`,
  `ErrInvalidNetwork`, and `ErrSimulationLogicError` exit 1.
  `ErrConfigFailed` and `ErrSimulatorNotFound` exit 2.
- Anything else exits 3.

Exit codes 4, 5, 6, and 143 are never returned by the Go binary. In
particular, the Go binary maps SIGTERM to 130, not 143.

### Go shell script example

```sh
glassbox debug --network testnet "$TX_HASH"
rc=$?
case $rc in
  0)   echo "Debug completed" ;;
  1)   echo "User error: check the hash, --network, or token" ;;
  2)   echo "Configuration error: run 'glassbox config show --explain' or 'glassbox doctor'" ;;
  3)   echo "Runtime error: RPC, simulator, or internal failure" ;;
  130) echo "Interrupted" ;;
  *)   echo "Unexpected exit code $rc" ;;
esac
```

### Verifying the Go table

From the repository root:

```sh
# Exit-code constants and the two code maps
sed -n '/^const (/,/^)/p' internal/cmd/exitcode.go
sed -n '/^var userErrorCodes/,/^}/p;/^var configErrorCodes/,/^}/p' internal/cmd/exitcode.go

# SIGTERM shares the 130 path with SIGINT
grep -n 'SIGINT/SIGTERM' internal/cmd/interrupt.go
```

Every code listed in the maps should appear in rows 1 and 2 above.

## TypeScript shell script example

```sh
glassbox audit:verify --file report.json
case $? in
  0) echo "Verified OK" ;;
  2) echo "Bad input — check your arguments" ;;
  5) echo "Signature invalid or expired" ;;
  *) echo "Unexpected error (code $?)" ;;
esac
```

`audit:verify` is implemented in the TypeScript wrapper (`src/commands/audit.ts`),
so it uses the TypeScript registry.

## Rules for contributors

1. **Always use the registry.**  Do not hard-code numeric literals; import
   `ExitCode` (TypeScript) or add the code to `userErrorCodes` /
   `configErrorCodes` in `internal/cmd/exitcode.go` (Go).
2. **Prefer specific codes over UNKNOWN_ERROR.**  If a failure falls into a
   documented category, use that category's code.
3. **SUCCESS must be zero.**  Any non-zero code is an error condition.
4. **Do not add codes without updating this file.**  New codes require a PR
   that adds the row here, updates `src/exit-codes.ts`, and adds a test.
