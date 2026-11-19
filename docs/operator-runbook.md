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
