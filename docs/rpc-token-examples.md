# RPC Bearer Token Examples

Closes #1301.

Some private Soroban RPC providers require an authentication token. Glassbox
sends it as an HTTP `Authorization: Bearer <token>` header on every RPC request.
This page shows how to supply the token safely, how to check that Glassbox
picked it up without printing it, and what the failure looks like when the
token is wrong.

All examples use `example-not-a-real-token` as the token value. Never paste a
real token into a shell command that ends up in your history, a CI log, or an
issue.

---

## How the token is resolved

Glassbox checks three sources, in this order (the first non-empty value wins):

| Priority | Source | Notes |
|---:|---|---|
| 1 | `--rpc-token <token>` flag | Only on commands that define the flag (see below). |
| 2 | `GLASSBOX_RPC_TOKEN` environment variable | Read by every RPC-backed command. |
| 3 | `rpc_token` key in the config file | See [`config show`](./config-show.md) for file locations. |

When a token is set, the RPC transport (`internal/rpc/http.go`) adds
`Authorization: Bearer <token>` to each request. When no token is set, the
header is not sent at all.

### Commands that accept `--rpc-token`

`debug`, `explain`, `dry-run`, `shell`, `compare`, `profile`, and
`debug bundle create`.

`auth-debug` has **no** `--rpc-token` flag. It reads the token from
`GLASSBOX_RPC_TOKEN` first and then from the config file `rpc_token` value
(`authResolveRPCToken` in `internal/cmd/auth_debug.go`).

---

## Supplying the token safely

### Environment variable (recommended)

Read the token without echoing it, then export it for the current shell only:

```sh
read -rs GLASSBOX_RPC_TOKEN   # type or paste the token; nothing is shown
export GLASSBOX_RPC_TOKEN
glassbox debug --network testnet <tx-hash>
```

In CI, map the token from the platform's secret store, not from a literal in the
workflow file:

```yaml
# GitHub Actions
env:
  GLASSBOX_RPC_TOKEN: ${{ secrets.GLASSBOX_RPC_TOKEN }}
```

### Config file

Add the key to your Glassbox config file, for example
`~/.glassbox/config.toml`, and restrict the file's permissions:

```toml
rpc_url   = "https://rpc.example-provider.invalid"
rpc_token = "example-not-a-real-token"
```

```sh
chmod 600 ~/.glassbox/config.toml
```

Do not put `rpc_token` in a project-local `.glassbox.toml` that is committed to
the repository.

### `--rpc-token` flag

The flag works, but the value shows up in shell history and in process listings
(`ps`). Use it only with a variable, never with a literal:

```sh
glassbox debug --rpc-token "$GLASSBOX_RPC_TOKEN" --network testnet <tx-hash>
```

### Avoid trailing newlines

A token read from a file with `$(cat file)` is fine, because command
substitution strips trailing newlines. A token piped in by other tools can keep a
trailing `\r` or `\n`, and the provider then rejects it. To check for that
without printing the token:

```sh
printf '%s' "$GLASSBOX_RPC_TOKEN" | tail -c 1 | od -c | head -1
# Expected: the last character is part of the token, not \n or \r
```

---

## Checking which token source is active (without printing it)

`glassbox config show` never prints the token. When one is set, the value is
always `[redacted]`, and the `source` field tells you where it came from:

```sh
GLASSBOX_RPC_TOKEN=example-not-a-real-token glassbox config show --json \
  | jq '.values.rpc_token'
```

Expected output:

```json
{
  "value": "[redacted]",
  "source": "environment"
}
```

`source` is `environment` when `GLASSBOX_RPC_TOKEN` is set, `file` when the
token came from a config file, and `default` otherwise
(`resolveConfigSource` in `internal/cmd/config_show.go`).

If no token is set from any source, the `rpc_token` key is absent from
`.values` and `jq` prints `null`.

`glassbox config show --explain` prints the same field with its precedence and
redaction status in text form.

To check that a token is set at all, test its length rather than printing any
of it:

```sh
[ -n "$GLASSBOX_RPC_TOKEN" ] && echo "token set (${#GLASSBOX_RPC_TOKEN} chars)" || echo "token not set"
```

---

## Failure path: rejected token

When the provider rejects the token, the RPC client classifies the HTTP 4xx
response and attaches this hint (`internal/rpc/failure_cause.go`):

```
The RPC server rejected the request (4xx). Check your authentication token
(--rpc-token / GLASSBOX_RPC_TOKEN) and confirm the endpoint supports the
requested method.
```

Errors classified as `UNAUTHORIZED` exit with code **1** (user error). See
[Stable error codes](./stable-error-codes.md) and
[Exit codes](./exit-codes.md).

To fix it, follow
[Operator runbook — Authentication errors](./operator-runbook.md#authentication-errors).

---

## Where the token is never written

| Surface | Behavior |
|---|---|
| `glassbox config show` / `--json` / `--explain` | Value replaced with `[redacted]` |
| Diagnostics bundle (`glassbox doctor --bundle`) | `config.rpc_token` replaced with `[REDACTED]`; `GLASSBOX_RPC_TOKEN` is on the env redaction list |
| Debug bundles | Bundles must not contain provider credentials (`internal/bundle/bundle.go`) |
| Redaction profiles | `rpc_token` is a `FieldToken` rule (`internal/redaction/profile.go`) |

---

## Verifying this page against the source

Run these from the repository root on a clean checkout. Each one should print
at least one matching line.

```sh
# Bearer header format
grep -n '"Bearer "' internal/rpc/http.go

# Env var read by auth-debug (no --rpc-token flag there)
grep -n 'GLASSBOX_RPC_TOKEN' internal/cmd/auth_debug.go
grep -c '"rpc-token"' internal/cmd/auth_debug.go   # expected: 0

# Commands that define --rpc-token
grep -ln '"rpc-token"' internal/cmd/*.go | grep -v _test

# config show redacts the token
grep -n -A2 'output.Values\["rpc_token"\]' internal/cmd/config_show.go

# 4xx remediation hint
grep -n 'GLASSBOX_RPC_TOKEN' internal/rpc/failure_cause.go
```
