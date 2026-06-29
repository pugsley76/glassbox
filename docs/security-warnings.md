# Contract Security Warnings

Glassbox emits proactive warnings for insecure Soroban patterns detected in
contract source or metadata.

Current contract-level checks include:

- Open authorization: privileged functions such as owner/admin/mint/update paths
  without an obvious `require_auth`, `require_auth_for_args`, or `check_auth`.
- Unchecked minting: token mint paths without both an authorization guard and a
  visible supply cap or policy check.
- Unsafe signature types: raw signature verification without obvious domain
  separation, nonce, or network binding.

Warnings are heuristic. Treat them as prompts for review, not proof of a bug.
Each warning includes severity, evidence, and a safer alternative.

## Severity levels

| Severity | Meaning | Recommended action |
|----------|---------|-------------------|
| `high` | Pattern strongly associated with exploitable vulnerabilities | Review immediately before deployment |
| `medium` | Pattern associated with common contract bugs | Review during normal development |
| `low` | Pattern that may reduce security posture in specific contexts | Consider reviewing before mainnet |
| `info` | Informational; not necessarily a problem | Awareness only |

## Operation audit log — security and privacy

### Sensitive value redaction

The operation audit log (`--audit-log`) redacts sensitive values before signing
so that tokens, keys, and passwords never appear in the signed record.

**Redaction rules:**

| Input location | Rule |
|----------------|------|
| CLI flag names containing `token`, `secret`, `password`, `private`, `key`, `pin`, or `passphrase` | Value replaced with `REDACTED` |
| Long (≥16 char) values that also contain hex characters | Value replaced with `REDACTED` (likely-secret heuristic) |
| Error messages containing file paths (`/…` or `C:\…`) | Path segments replaced with `<path>` |
| Error messages containing likely-secret values | Value replaced with `REDACTED` |

Both long-form (`--rpc-token value`, `--rpc-token=value`) and short-form
(`-t value`) flag values are covered by the redaction pass.

### Metadata entry validation

User-supplied `--metadata key=value` entries are validated before being
embedded in the signed payload:

| Condition | Action |
|-----------|--------|
| Key is empty or exceeds 128 characters | Entry skipped |
| Value exceeds 1024 characters | Value truncated with `… (truncated)` marker |
| Key or value contains null bytes (`\x00`) | Entry skipped |
| Entry has no `=` separator | Entry skipped |

Validation errors are silent (entries are skipped rather than causing a fatal
error) so that a malformed metadata flag does not abort an otherwise valid
audit log write.

## Telemetry privacy

### Command name sanitization

Before the command name is emitted to the OTLP collector, it is sanitized to
contain only alphanumeric characters, dashes, colons, and underscores — the
characters used by all built-in glassbox commands. Any other character is
replaced with `_` and the name is capped at 64 characters. This prevents
user-controlled input (e.g. a malformed deep-link subcommand) from leaking
arbitrary data to the telemetry backend.

### Value fingerprinting

Telemetry attribute values for keys containing `hash`, `tx`, or `contract` are
replaced with a **32-character fingerprint** (`sha256:<hex_prefix>`). This is
intentionally lossy — the full value cannot be recovered from the fingerprint,
and it is suitable only for telemetry aggregation, not cryptographic purposes.

### Config show — URL credential stripping

`glassbox config show` strips `user:password@` userinfo from URLs before
displaying `soroban_rpc_urls`. URLs that cannot be parsed are replaced with
`[invalid url]`. The JSON output path and the human-readable path both apply
this stripping.

The Go API is:

```go
detector := security.NewDetector()
findings := detector.AnalyzeContractSource(security.SourceContext{
    Path: "contract/src/lib.rs",
    Source: sourceCode,
    Metadata: metadata,
})
```
