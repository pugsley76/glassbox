# Troubleshooting Index: Setup Error Codes

A lookup from every **setup** error code to its cause, the `glassbox doctor`
check that detects it, and the fix. "Setup" means the code reports a problem
with the local environment, configuration, network selection, RPC endpoint,
or credentials, not with the transaction being debugged.

Codes, exit codes, and messages come from `internal/errors/`,
`internal/cmd/exitcode.go`, `internal/config/`, and `internal/simulator/runner.go`.
The full catalogue lives in [stable-error-codes.md](stable-error-codes.md);
the triage procedure is in [operator-runbook.md](operator-runbook.md).

Errors print to stderr as `Error: <CODE>: <message>` (see
[stable-error-codes.md](stable-error-codes.md#what-the-cli-process-prints-today)).

## Lookup

| Code | Exit | Typical message | `doctor` check | Section |
|---|---|---|---|---|
| `SIMULATOR_NOT_FOUND` | 2 | `glassbox-sim binary not found (use --sim-path or set GLASSBOX_SIM_PATH)` | Simulator Binary (glassbox-sim) | [↓](#simulator_not_found) |
| `CONFIG_ERROR` | 2 | `failed to read config file` / `failed to parse config file` | TOML config | [↓](#config_error) |
| `INVALID_NETWORK` | 1 | `invalid network "<name>" — must be one of: testnet, mainnet, futurenet` | — | [↓](#invalid_network) |
| `NETWORK_NOT_FOUND` | 1 | `<name>` | — | [↓](#network_not_found) |
| `RPC_CONNECTION_FAILED` | 3 | `RPC connection failed: <cause>` | RPC endpoint | [↓](#rpc_connection_failed-rpc_timeout-all_rpc_failed) |
| `RPC_TIMEOUT` | 3 | `RPC request timed out: <cause>` | RPC endpoint | [↓](#rpc_connection_failed-rpc_timeout-all_rpc_failed) |
| `ALL_RPC_FAILED` | 3 | `all RPC endpoints failed` | RPC endpoint | [↓](#rpc_connection_failed-rpc_timeout-all_rpc_failed) |
| `RATE_LIMIT_EXCEEDED` | 1 | `rate limit exceeded, please try again later` | — | [↓](#rate_limit_exceeded) |
| `UNAUTHORIZED` | 1 | `unauthorized` | — | [↓](#unauthorized) |

`glassbox doctor` checks that do not map to an error code (Go, Rust, Cargo,
cache directory, protocol registry, Go module dependencies, deep link) are
covered in [operator-runbook.md](operator-runbook.md#doctor-checks-without-an-error-code).

## SIMULATOR_NOT_FOUND

**Cause.** No executable `glassbox-sim` was found. The runner searches, in
order: `--sim-path`, `GLASSBOX_SIM_PATH`, `./glassbox-sim`, `./bin/glassbox-sim`,
`simulator/target/debug/glassbox-sim`, `simulator/target/release/glassbox-sim`,
then `PATH`. When `--sim-path` points at a missing or non-executable file, the
message is that path.

**Detect.** `glassbox doctor` → `[FAIL] Simulator Binary (glassbox-sim)`.

**Fix.**

```bash
cd simulator && cargo build --release
# or point at an existing binary
export GLASSBOX_SIM_PATH=/path/to/glassbox-sim
```

## CONFIG_ERROR

**Cause.** A config file could not be read, parsed, or written: the JSON
config at `~/.Glassbox/config.json`, a TOML config (`.Glassbox.toml`,
`~/.Glassbox.toml`, `/etc/Glassbox/config.toml`), or the custom network config
under `~/.Glassbox/`. Messages raised by `internal/config`:

- `failed to read config file`
- `failed to parse config file`
- `failed to create config directory`
- `failed to marshal config`
- `failed to get home directory`
- `schema_version must be a non-negative integer`
- `config file declares schema_version <n> but this Glassbox binary only supports up to version <m>; upgrade Glassbox or run 'glassbox config migrate'`

`WrapConfigError` does not keep the underlying cause, so the message does not
say *why* the read or parse failed. Inspect the file directly.

**Detect.** `glassbox doctor` → `[FAIL] TOML config` reports the failing line
(`line <n> missing '='` or `line <n> has invalid key=value`) for the three TOML
paths above. It does not check `~/.Glassbox/config.json`.
`glassbox config show` prints the paths that are loaded (see
[config-show.md](config-show.md)).

**Fix.** Correct or remove the malformed file. For a newer `schema_version`,
upgrade Glassbox or run `glassbox config migrate`.

## INVALID_NETWORK

**Cause.** `--network` is not `testnet`, `mainnet`, `futurenet`, or a saved
custom network. Raised by `shell`, `search`, `compare`, `daemon`, `bundle`,
`offline`, and config validation.

> `glassbox debug` validates `--network` itself and reports
> `VALIDATION_FAILED` (`invalid --network "<name>"; must be one of: testnet, mainnet, futurenet`)
> instead of `INVALID_NETWORK`. Both exit `1`; match both codes in automation.

**Fix.** Use one of the built-in networks, or register the custom network
before referencing it.

## NETWORK_NOT_FOUND

**Cause.** A named custom network has no saved configuration in
`~/.Glassbox/`. The message is only the network name.

**Fix.** Add the network to the custom network config, or use a built-in
network.

## RPC_CONNECTION_FAILED, RPC_TIMEOUT, ALL_RPC_FAILED

**Cause.** The configured RPC endpoint is unreachable, too slow, or every
failover endpoint failed. `RPC_CONNECTION_FAILED` and `RPC_TIMEOUT` include a
`Hint:` line on stderr.

**Detect.** `glassbox doctor` → `[FAIL] RPC endpoint` with
`RPC endpoint "<url>" is unreachable — set GLASSBOX_RPC_URL or check your network connection`.

**Fix.**

```bash
export GLASSBOX_RPC_URL=<reachable soroban rpc url>
# or per command
glassbox debug <tx-hash> --network testnet --rpc-url <url>
```

## RATE_LIMIT_EXCEEDED

**Cause.** The RPC provider throttled requests.

**Fix.** Retry later or switch to an endpoint with a higher limit via
`--rpc-url` or `GLASSBOX_RPC_URL`.

## UNAUTHORIZED

**Cause.** A request to `glassbox daemon` did not send
`Authorization: Bearer <token>` matching the `--auth-token` the daemon was
started with.

**Fix.** Send the same token the daemon was started with
(`glassbox daemon --auth-token <token>`).

## Verifying this index

```bash
# Code constants and wrapper messages
go test ./internal/errors/
# Code → exit code mapping (ExitCodeFor)
go test ./internal/cmd/ -run 'ExitCode|ErstError'
# Error standardization checks
bash scripts/validate-errors.sh
```
