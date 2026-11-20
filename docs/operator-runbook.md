# Operator Runbook: Setup Triage

Use this procedure when a Glassbox command fails before it can analyze a
transaction. Look up each error code in
[troubleshooting-index.md](troubleshooting-index.md).

## 1. Capture the failure

```bash
glassbox <command> ... 2> err.txt
echo "exit=$?"
sed -n 's/^Error: \([A-Z_]*\): .*/\1/p' err.txt | head -n1
```

| Exit | Meaning | Next step |
|---|---|---|
| 1 | User input | Fix the flag or argument named in the message |
| 2 | Config / environment | Go to step 2; the code is `CONFIG_ERROR` or `SIMULATOR_NOT_FOUND` |
| 3 | Internal / runtime | If the code is `RPC_*` or `ALL_RPC_FAILED`, go to step 2; otherwise file a bug with the bundle from step 3 |
| 130 | Interrupted | Re-run |

An `Error:` line without a code is a plain error. It still exits `3`; read
the message.

## 2. Run the environment check

```bash
glassbox doctor
glassbox doctor --verbose     # adds resolved paths
```

Each line is `[OK]`, `[FAIL]`, or `[FAIL*]` (fixable), followed by a `→` hint
on failure. For fixable items:

```bash
glassbox doctor --fix         # prompts before each fix
glassbox doctor --fix --yes   # non-interactive (CI)
```

> **Automation note:** `glassbox doctor` exits `0` even when checks fail. It
> reports problems in its output, not through its exit status. In CI, check
> for `[FAIL` in the output:
>
> ```bash
> glassbox doctor | tee doctor.txt
> ! grep -q '\[FAIL' doctor.txt
> ```

### Doctor checks with an error code

| Doctor check | Error code |
|---|---|
| Simulator Binary (glassbox-sim) | [`SIMULATOR_NOT_FOUND`](troubleshooting-index.md#simulator_not_found) |
| TOML config | [`CONFIG_ERROR`](troubleshooting-index.md#config_error) |
| RPC endpoint | [`RPC_CONNECTION_FAILED` / `RPC_TIMEOUT` / `ALL_RPC_FAILED`](troubleshooting-index.md#rpc_connection_failed-rpc_timeout-all_rpc_failed) |

### Doctor checks without an error code

These don't map to a stable code. Use the `→` hint `doctor` prints:

| Check | Hint printed on failure |
|---|---|
| Go | `Install Go from https://go.dev/doc/install (requires Go 1.21+)`, or `go.mod requests <v> but installed <v>` |
| Rust (rustc) | `Install Rust from https://rustup.rs/ ...` |
| Cargo | `Cargo is included with Rust. Install from https://rustup.rs/` |
| Cache directory (~/.Glassbox) | `Run 'Glassbox doctor --fix' to create cache directory` |
| Protocol Registry | `Run 'Glassbox doctor --fix' to initialize protocol registry` |
| Go Module Dependencies | `Run 'Glassbox doctor --fix' to resolve dependencies` |
| Deep link (glassbox:// scheme) | Platform-specific steps, e.g. `Try: Glassbox install-scheme` |

## 3. Escalate with a diagnostics bundle

```bash
glassbox doctor --bundle
glassbox doctor --bundle --bundle-output ./glassbox-diag.gbdiag
```

The archive is redacted (no private keys or tokens); see
[diagnostics-bundle.md](diagnostics-bundle.md). Attach it along with
`err.txt` and the exit code from step 1.

## 4. Saved sessions

If the failure involves `glassbox session resume` / `load`, run
`glassbox session doctor` and see
[session-bookmarking.md](session-bookmarking.md#failure-paths).
# Operator Runbook: RPC and Simulator Failures

This runbook organises common failures by **symptom**, lists the **evidence to
collect**, provides **safe remediation** steps, and indicates **when to generate
a diagnostics bundle** for escalation.

> **Principle:** Every fix in this runbook is **safe** — it does not modify your
> contract, ledger state, or deployment. If a step would require modification,
> it is marked as "escalate" instead.

---

## Table of Contents

1. [RPC connection failures](#rpc-connection-failures)
2. [RPC failover recovery](#rpc-failover-recovery)
3. [RPC timeouts](#rpc-timeouts)
4. [Transaction not found](#transaction-not-found)
5. [Rate limiting](#rate-limiting)
6. [Authentication errors](#authentication-errors)
7. [Simulator binary not found](#simulator-binary-not-found)
8. [Simulator crashes](#simulator-crashes)
9. [Simulation logic errors](#simulation-logic-errors)
10. [Source mapping failures](#source-mapping-failures)
11. [Protocol version mismatch](#protocol-version-mismatch)
12. [Snapshot corruption](#snapshot-corruption)
13. [Session integrity failures](#session-integrity-failures)
14. [Export failures](#export-failures)
15. [Diagnostics bundle](#generating-a-diagnostics-bundle)

---

## RPC connection failures

**Stable error code:** `RPC_CONNECTION_FAILED`

**Symptoms:**
- `glassbox debug` fails immediately with `RPC connection failed: dial tcp ...`
- All failover endpoints report the same error

**Evidence to collect:**

```sh
# Test basic connectivity
curl -s -o /dev/null -w "%{http_code}" https://soroban-testnet.stellar.org

# Check DNS resolution
nslookup soroban-testnet.stellar.org

# Verify Glassbox config
glassbox config show --json | jq '.rpc_url'
```

**Safe remediation:**

1. **Check internet connection** — open a browser or `curl` a known URL.
2. **Try an alternative endpoint** — pass `--rpc-url` with a different endpoint:
   ```sh
   glassbox debug --rpc-url https://rpc.stellar.org --network testnet <tx-hash>
   ```
3. **Check firewall rules** — ensure outbound HTTPS (443) is allowed.
4. **Verify RPC token** — if using a private endpoint, confirm a token is set
   without printing any of it:
   ```sh
   [ -n "$GLASSBOX_RPC_TOKEN" ] && echo "token set (${#GLASSBOX_RPC_TOKEN} chars)" || echo "token not set"
   ```
   See [RPC bearer token examples](./rpc-token-examples.md).

**When to escalate:** If the endpoint is reachable via `curl` but Glassbox
cannot connect, generate a diagnostics bundle:

```sh
glassbox doctor --bundle --bundle-output ./diag-connection.gbdiag
```

---

## RPC failover recovery

**Stable error code:** `ALL_RPC_FAILED`

**Symptoms:**
- `glassbox debug` fails with `all RPC endpoints failed` after trying multiple endpoints
- The error output includes a `Provider failover summary` block listing each attempted endpoint and its failure cause
- `glassbox rpc health` shows multiple endpoints with `[FAIL]` status

**Evidence to collect:**

```sh
# Check configured endpoints
glassbox config show --json | jq '.soroban_rpc_urls // .rpc_urls // .rpc_url'

# Probe each endpoint's health
glassbox rpc health

# Check failover strategy
glassbox config show --json | jq '.failover_strategy'

# Check circuit breaker settings
glassbox config show --json | jq '.failure_threshold, .retry_timeout'
```

**Safe remediation:**

1. **Verify endpoint configuration** — ensure `soroban_rpc_urls` (or `rpc_urls` /
   `rpc_url`) contains valid, reachable endpoints. Multiple endpoints enable
   automatic failover.

   ```toml
   # glassbox.toml
   soroban_rpc_urls = [
     "https://soroban-testnet.stellar.org",
     "https://rpc.example.com"
   ]
   failover_strategy = "weighted"  # or "sticky", "round_robin"
   failure_threshold = 5           # consecutive failures before circuit opens
   retry_timeout = 60              # seconds before re-probing a failed endpoint
   ```

   Or via environment:
   ```sh
   export GLASSBOX_SOROBAN_RPC_URLS="https://endpoint1.example.com,https://endpoint2.example.com"
   export GLASSBOX_FAILOVER_STRATEGY="weighted"
   ```

2. **Check individual endpoint health** — use `glassbox rpc health` to probe each
   configured endpoint and see latency/status:

   ```sh
   glassbox rpc health
   # or with explicit URLs:
   glassbox rpc health --rpc "https://endpoint1.example.com,https://endpoint2.example.com"
   ```

3. **Adjust failover strategy** — if traffic should prefer the fastest endpoint,
   use `sticky`; for even distribution, use `round_robin`; for health-weighted
   selection, use `weighted` (default).

4. **Tune circuit breaker** — if endpoints recover slowly, increase
   `retry_timeout`; if you want faster failover, decrease `failure_threshold`.

5. **Pin a specific endpoint** — for reproducible debugging, use `--pin-endpoint`
   to lock to a single RPC endpoint (disables silent failover):

   ```sh
   glassbox debug --pin-endpoint https://soroban-testnet.stellar.org --network testnet <tx-hash>
   ```

   Note: `--pin-endpoint` must match `--rpc-url` when both are provided.

**When to escalate:** If all configured endpoints are healthy per `rpc health`
but `debug` still fails, generate a diagnostics bundle:

```sh
glassbox doctor --bundle --bundle-output ./diag-failover.gbdiag
```

---

## RPC timeouts

**Stable error code:** `RPC_TIMEOUT`

**Symptoms:**
- Debug run hangs for > 10 seconds then fails with timeout error
- `--show-metrics` shows high `rpc_avg_ms`

**Evidence to collect:**

```sh
# Measure RPC latency
time curl -s -X POST https://soroban-testnet.stellar.org \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"getHealth"}'

# Check Glassbox metrics
glassbox debug --show-metrics --network testnet <tx-hash>
```

**Safe remediation:**

1. **Check RPC endpoint health:**
   ```sh
   curl -s https://soroban-testnet.stellar.org | jq .
   ```
2. **Use a closer endpoint** — if on a different continent, try a regional
   endpoint.
3. **Increase timeout** — set `request_timeout` in `glassbox.toml` (top-level,
   not under `[rpc]`):
   ```toml
   request_timeout = 30  # seconds (default: 15)
   ```
   Or set the environment variable:
   ```sh
   export GLASSBOX_REQUEST_TIMEOUT=30
   ```
4. **Check for slow calls** — `--show-metrics` highlights calls > 3 seconds
   with a remediation tip.

**When to escalate:** If latency exceeds 5 seconds consistently, file an issue
with the `--show-metrics` JSON output attached.

**Verification:**

```sh
# Check current timeout configuration
glassbox config show --json | jq '.request_timeout'

# Test RPC latency manually
time curl -s -X POST https://soroban-testnet.stellar.org \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"getHealth"}'

# Run debug with metrics to see RPC performance
glassbox debug --show-metrics --network testnet <tx-hash>
```

**Test coverage:**
- `internal/rpc/client_test.go` — `TestWithRequestTimeout_*` tests verify timeout configuration
- `internal/rpc/failure_cause_test.go` — `TestClassifyFailureCause_*` tests verify timeout error classification
- `internal/config/config_test.go` — `TestDefaultConfig_RequestTimeout`, `TestLoad_RequestTimeout*` tests verify config parsing

---

## Transaction not found

**Stable error code:** `TRANSACTION_NOT_FOUND`

**Symptoms:**
- `transaction not found` error after successful RPC connection

**Evidence to collect:**

```sh
# Verify hash format (64 lowercase hex chars)
echo "<tx-hash>" | wc -c  # should be 65 (64 + newline)

# Check the correct network
glassbox debug --network testnet <tx-hash>
glassbox debug --network mainnet <tx-hash>

# Verify the transaction exists on the network
curl -s "https://soroban-testnet.stellar.org/transactions/<tx-hash>" | jq .
```

**Safe remediation:**

1. **Verify hash** — ensure 64 lowercase hexadecimal characters.
2. **Check network** — the transaction may be on a different network. Use
   `--network testnet`, `--network mainnet`, or `--network futurenet`.
3. **Wait for propagation** — newly submitted transactions may take a few
   seconds to appear. Use `--watch` to poll:
   ```sh
   glassbox debug --watch --watch-timeout 60 --network testnet <tx-hash>
   ```
4. **Check confirmation** — some RPC providers require confirmation before
   returning transaction data.

**When to escalate:** If the transaction is confirmed on the network explorer
but Glassbox cannot find it, generate a diagnostics bundle.

---

## Rate limiting

**Stable error code:** `RATE_LIMIT_EXCEEDED`

**Symptoms:**
- `rate limit exceeded` error from RPC
- Multiple rapid requests fail

**Evidence to collect:**

```sh
# Check current rate limit headers (if returned by endpoint)
curl -s -D - -X POST https://soroban-testnet.stellar.org \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"getHealth"}' | grep -i "ratelimit\|retry-after"
```

**Safe remediation:**

1. **Wait and retry** — respect the `Retry-After` header if present.
2. **Reduce request frequency** — avoid running multiple `glassbox debug`
   sessions simultaneously against the same endpoint.
3. **Use a private endpoint** — if available, configure `--rpc-url` with a
   dedicated endpoint.
4. **Enable caching** — use `--no-cache=false` (default) to leverage local
   ledger state caching.

**When to escalate:** Persistent rate limiting with low request volume indicates
a misconfiguration. Generate a diagnostics bundle.

---

## Authentication errors

**Stable error code:** `UNAUTHORIZED`

**Symptoms:**
- `unauthorized` or `missing/invalid auth token` error

**Evidence to collect:**

```sh
# Check a token is set without printing any of it
[ -n "$GLASSBOX_RPC_TOKEN" ] && echo "token set (${#GLASSBOX_RPC_TOKEN} chars)" || echo "token not set"

# Check which source supplied the token (value is always "[redacted]")
glassbox config show --json | jq '.values.rpc_token'
# Expected: {"value": "[redacted]", "source": "environment"}  (or "file")
# null means no token is configured from any source
```

**Safe remediation:**

1. **Verify token** — ensure `GLASSBOX_RPC_TOKEN` is set and not expired.
2. **Re-authenticate** — generate a new token from your RPC provider dashboard.
3. **Check token scope** — some tokens are network-specific (testnet vs
   mainnet).

**When to escalate:** If the token is valid for other tools but rejected by
Glassbox, check for extra whitespace or newline characters in the environment
variable. Glassbox sends the value unchanged as `Authorization: Bearer <token>`.
See [RPC bearer token examples](./rpc-token-examples.md) for safe ways to set
and check the token.

---

## Simulator binary not found

**Stable error code:** `SIMULATOR_NOT_FOUND`

**Symptoms:**
- `simulator binary not found` error
- `glassbox doctor --fix` suggests building the simulator

**Evidence to collect:**

```sh
# Check if simulator exists
which glassbox-sim 2>/dev/null || echo "not found"

# Check Glassbox doctor
glassbox doctor --verbose
```

**Safe remediation:**

1. **Run doctor --fix:**
   ```sh
   glassbox doctor --fix
   ```
   In an interactive terminal, Glassbox asks before each fix. In CI or another
   non-interactive environment, use `--yes` only when the job is authorized to
   apply those fixes; without it, the command returns an error instead of
   accepting the prompts automatically:
   ```sh
   glassbox --non-interactive doctor --fix --yes
   ```
2. **Build manually:**
   ```sh
   cd simulator && cargo build --release
   ```
3. **Set explicit path** — if the binary is in a non-standard location:
   ```toml
   # glassbox.toml
   [simulator]
   path = "/path/to/glassbox-sim"
   ```

**When to escalate:** If `cargo build` fails, check Rust toolchain version
(`rustc --version`) and ensure the `wasm32-unknown-unknown` target is installed:
```sh
rustup target add wasm32-unknown-unknown
```

---

## Simulator crashes

**Stable error code:** `SIMULATOR_CRASH`

**Symptoms:**
- Process exits unexpectedly with signal (SIGSEGV, SIGABRT)
- `simulator process crashed unexpectedly` error

**Evidence to collect:**

```sh
# Run with verbose logging
glassbox debug --verbose --network testnet <tx-hash>

# Check simulator version
glassbox-sim --version 2>/dev/null || echo "version unknown"

# Generate diagnostics bundle
glassbox doctor --bundle --bundle-output ./diag-crash.gbdiag
```

**Safe remediation:**

1. **Rebuild the simulator** — crashes may be caused by stale binaries:
   ```sh
   cd simulator && cargo build --release
   ```
2. **Check memory limits** — large contracts may exceed available memory.
   Monitor with `top` or `htop` during execution.
3. **Reduce trace depth** — if the contract has deep call stacks:
   ```sh
   glassbox debug --trace-verbosity summary --network testnet <tx-hash>
   ```

**When to escalate:** Generate a diagnostics bundle and file an issue with the
crash signal and stack trace.

---

## Simulation logic errors

**Stable error code:** `SIMULATION_LOGIC_ERROR`

**Symptoms:**
- Simulation completes but reports a contract logic error
- `simulation execution failed: <detail>`

**Evidence to collect:**

```sh
# Full trace output
glassbox debug --trace-verbosity verbose --json --network testnet <tx-hash> | jq .

# Check budget usage
glassbox debug --show-metrics --network testnet <tx-hash>

# Decode the error
glassbox debug --format json --network testnet <tx-hash> | jq '.result.error'
```

**Safe remediation:**

1. **Check the error code** — stable error codes are listed in
   [stable-error-codes.md](./stable-error-codes.md).
2. **Review the trace** — look at the `states` array for the failing step and
   its `source_location`.
3. **Check budget** — if `cpu_usage_percent` or `memory_usage_percent` is near
   100%, the contract hit resource limits.
4. **Verify arguments** — ensure `--args` match the contract's expected
   parameters.
5. **Compare with network** — run with `--compare-network` to see if the
   failure reproduces differently on another network.

**When to escalate:** If the simulation fails but the same transaction succeeds
on-chain, the issue may be a snapshot or ledger state mismatch. Generate a
diagnostics bundle with `--save-snapshots`.

---

## Source mapping failures

**Stable error code:** `SOURCE_DISCOVERY_FAILED`

**Symptoms:**
- `contract source not found: all discovery stages exhausted`
- Source locations show `unknown` or WAT disassembly

**Evidence to collect:**

```sh
# Dry-run source discovery
glassbox debug --dry-run --network testnet <tx-hash>

# Check WASM binary
file <path-to-wasm>
wasm-objdump -h <path-to-wasm> | grep debug

# Verify source path
glassbox debug --wasm <path> --contract-source ./src --dry-run <tx-hash>
```

**Safe remediation:**

1. **Provide explicit source path:**
   ```sh
   glassbox debug --wasm ./contract.wasm --contract-source ./src <tx-hash>
   ```
2. **Use source alias** for path remapping:
   ```sh
   glassbox debug --source-alias ./aliases.json --wasm ./contract.wasm <tx-hash>
   ```
3. **Skip source mapping** if only raw trace is needed:
   ```sh
   glassbox debug --skip-source-mapping --wasm ./contract.wasm <tx-hash>
   ```

See [source-mapping-troubleshooting.md](./source-mapping-troubleshooting.md)
for the full troubleshooting wizard.

---

## Protocol version mismatch

**Stable error code:** `SIMULATION_FAILED` (with protocol version context)

**Symptoms:**
- `unsupported protocol version` error
- Simulation fails with version-related diagnostic

**Evidence to collect:**

```sh
# Check Glassbox version
glassbox version --json

# Check network protocol
glassbox config show --json | jq '.network'
```

**Safe remediation:**

1. **Update Glassbox** — download the latest release for protocol support.
2. **Override protocol version** (if you know the correct version):
   ```sh
   glassbox debug --protocol-version <n> --network testnet <tx-hash>
   ```
3. **Check network compatibility** — `futurenet` may support protocols not yet
   on `testnet`.

**When to escalate:** If the protocol version is supported but simulation still
fails, file an issue with the `glassbox version --json` output.

---

## Snapshot corruption

**Stable error code:** `SIMULATION_FAILED` (with snapshot context)

**Symptoms:**
- `snapshot fingerprint mismatch` error
- `snapshot tx hash mismatch` or `snapshot network mismatch`
- `snapshot is stale` error

**Evidence to collect:**

```sh
# Inspect the snapshot registry
cat <registry-file> | jq .

# Check stored hash vs computed
glassbox debug --load-snapshots <registry-file> --verbose <tx-hash>
```

**Safe remediation:**

1. **Regenerate snapshots** — delete the old file and re-capture:
   ```sh
   glassbox debug --save-snapshots ./registry.json --network testnet <tx-hash>
   ```
2. **Check staleness** — CLI parameters must match between capture and replay.
   Re-capture with the same flags.
3. **Verify network** — snapshots are network-specific. A testnet snapshot
   cannot replay on mainnet.

**When to escalate:** If freshly captured snapshots fail immediately, file an
issue with the registry JSON attached.

---

## Session integrity failures

**Symptoms:**
- `Session integrity check FAILED` on `session resume`
- Checkpoint validation errors after a crash

**Evidence to collect:**

```sh
# List sessions
glassbox session list

# Attempt recovery
glassbox session recover
```

**Safe remediation:**

1. **Recover from crash:**
   ```sh
   glassbox session recover
   ```
   If recovery fails, the checkpoint is corrupt — proceed to step 2.
2. **Delete corrupt session:**
   ```sh
   glassbox session delete <session-id>
   ```
3. **Re-run debug:**
   ```sh
   glassbox debug --network testnet <tx-hash>
   ```

See [debug-command.md](./debug-command.md#session-recovery-and-integrity) for
full integrity field validation details.

---

## Export failures

**Symptoms:**
- `--trace-output` fails to write
- Exported file is empty or malformed
- SVG export fails

**Evidence to collect:**

```sh
# Check disk space
df -h .

# Check write permissions
touch ./test-write && rm ./test-write

# Validate trace output path
glassbox debug --dry-run --trace-output ./traces/debug.json --network testnet <tx-hash>
```

**Safe remediation:**

1. **Verify disk space** — ensure sufficient space for the output file.
2. **Check permissions** — the output directory must be writable.
3. **Use absolute paths** — relative paths may resolve unexpectedly:
   ```sh
   glassbox debug --trace-output /absolute/path/to/output.json --network testnet <tx-hash>
   ```
4. **Validate output** — after export, verify the file:
   ```sh
   jq . <output-file>  # should parse without error
   ```

**When to escalate:** If the directory is writable but export still fails,
generate a diagnostics bundle.

---

## Plugin policy blocked (untrusted / community plugins)

**Symptoms:**
- Error message: `plugin "..." has trust level "community" (source: ...) which is not allowed by policy`
- Plugins fail to load at startup or during `glassbox plugin list` / decoding.

**Evidence to collect:**

```sh
# List plugins and check their trust levels
glassbox plugin list
```

**Safe remediation:**
By default, Glassbox blocks community and untrusted plugins (`AllowUntrusted = false`) to protect against supply-chain compromises. To allow them explicitly:
1. **Pass `--allow-untrusted-plugins` on the CLI:**
   ```sh
   glassbox --allow-untrusted-plugins plugin list
   ```
2. **Or configure in `glassbox.toml`:**
   ```toml
   [plugin]
   allow_untrusted = true
   ```
3. **Or set environment variable:**
   ```sh
   export GLASSBOX_PLUGIN_ALLOW_UNTRUSTED=true
   ```

---

## Generating a diagnostics bundle

When a fix cannot be determined from the steps above, generate a portable
diagnostics archive:

```sh
# Standard bundle (auto-named in temp directory)
glassbox doctor --bundle

# Named bundle with verbose output
glassbox doctor --verbose --bundle --bundle-output ./diag.gbdiag

# Include in an issue or PR
gh issue create --title "Bug: <description>" \
  --body "Diagnostics: $(glassbox doctor --bundle --bundle-output ./diag.gbdiag)" \
  --repo pugsley76/glassbox
```

The bundle is automatically redacted — no private keys, tokens, or sensitive
data are included. See [diagnostics-bundle.md](./diagnostics-bundle.md) for
details.

---

## Offline reproduction

For reproducible offline debugging:

1. **Capture a snapshot:**
   ```sh
   glassbox debug --save-snapshots ./repro-registry.json --network testnet <tx-hash>
   ```
2. **Replay without network:**
   ```sh
   glassbox debug --load-snapshots ./repro-registry.json
   ```
3. **Share the snapshot** — it contains everything needed to reproduce the issue
   locally.

---

## Quick reference: error codes to symptoms

| Error code | Likely cause | First step |
|-----------|-------------|------------|
| `RPC_CONNECTION_FAILED` | Network or endpoint issue | Check internet, try `--rpc-url` |
| `RPC_TIMEOUT` | Slow endpoint or network | Check `--show-metrics`, try alternative endpoint |
| `TRANSACTION_NOT_FOUND` | Wrong network or hash | Verify network and hash format |
| `RATE_LIMIT_EXCEEDED` | Too many requests | Wait, reduce frequency, use private endpoint |
| `UNAUTHORIZED` | Invalid or expired token | Re-authenticate with provider |
| `SIMULATOR_NOT_FOUND` | Missing binary | `glassbox doctor --fix` |
| `SIMULATOR_CRASH` | Binary or memory issue | Rebuild simulator, reduce trace depth |
| `SIMULATION_LOGIC_ERROR` | Contract or state issue | Check trace, budget, arguments |
| `SOURCE_DISCOVERY_FAILED` | Missing source or symbols | Provide `--contract-source` |
| `ALL_RPC_FAILED` | All endpoints down | Check `glassbox rpc health`, verify `soroban_rpc_urls` config |

## See also

- [Troubleshooting index — Doctor symptom-to-check mapping](./troubleshooting-index.md#doctor-symptom-to-check-mapping) — complete mapping of symptoms to `glassbox doctor` checks
- [Debug command](./debug-command.md) — full command reference
- [Source mapping troubleshooting](./source-mapping-troubleshooting.md) — source mapping wizard
- [Diagnostics bundle](./diagnostics-bundle.md) — portable diagnostics
- [CI artifacts](./ci-artifacts.md) — reproducing CI failures
- [Stable error codes](./stable-error-codes.md) — error code catalogue
- [Sandboxed replay](./sandboxed-replay.md) — offline debugging
