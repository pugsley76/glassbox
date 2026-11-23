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

The diagnostics bundle (`glassbox doctor --bundle`) applies an additional layer
of environment-variable redaction. `GLASSBOX_RPC_TOKEN`,
`GLASSBOX_SENTRY_DSN`, `GLASSBOX_CONFIG_PASSPHRASE`, and
`GLASSBOX_CRASH_ENDPOINT` are always replaced with `[REDACTED]` regardless of
their content (`internal/diagnostics/redact.go: secretEnvKeys`).

The `session share` command uses a separate, configurable redaction system
(`internal/redaction/profile.go`). Two built-in profiles are provided:

| Profile name | What it redacts |
|---|---|
| `full` (default) | Secrets, tokens, account IDs, contract IDs, paths, PINs |
| `secrets-only` | Only secrets, tokens, and credentials; account/contract IDs and paths are preserved |

Select a profile with `--redact <profile-name>`. Use `--preview` to inspect
what would be removed before writing the archive. Custom profiles can be loaded
from a JSON file with `LoadProfileFromFile` (`internal/redaction/profile.go`).

#### Verifying redaction behaviour from a clean checkout

**Success path — audit log redaction:**

```sh
# A flag whose name contains "token" is redacted in the signed record.
# Run from the repository root; no network required.
glassbox audit:sign \
  --signing-provider software \
  --software-private-key "$(openssl genpkey -algorithm ed25519 2>/dev/null)" \
  --rpc-token example-not-a-real-token \
  --payload '{"input":{},"state":{},"events":[],"timestamp":"2026-01-01T00:00:00.000Z"}' \
  | python3 -m json.tool | grep -A2 '"args"'
# Expected: "REDACTED" appears in place of the token value; the flag name is kept.
```

**Success path — config show redaction (lowercase `[redacted]`):**

```sh
GLASSBOX_RPC_TOKEN=example-not-a-real-token glassbox config show --json \
  | python3 -m json.tool | grep -A3 rpc_token
# Expected output:
# "rpc_token": {
#   "value": "[redacted]",
#   "source": "environment"
# }
```

Note: `config show` uses `[redacted]` (lowercase). The audit log and
diagnostics bundle use `REDACTED` and `[REDACTED]` respectively — these are
different surfaces with different placeholders by design.

**Failure path — running `config show` without a token set:**

```sh
glassbox config show --json | python3 -m json.tool | grep rpc_token
# Expected: no output (the key is absent from .values when no token is set)
```

**Verifying source constants:**

```sh
# Audit log placeholder ("REDACTED")
grep -n '"REDACTED"' internal/cmd/operation_audit.go

# Session-share / report placeholder ("[REDACTED]")
grep -n 'Placeholder' internal/redaction/profile.go | head -3

# Diagnostics bundle placeholder ("[REDACTED]")
grep -n 'RedactedPlaceholder' internal/diagnostics/redact.go | head -1

# secretEnvKeys list (always-redacted env vars in doctor bundle)
grep -A8 'secretEnvKeys' internal/diagnostics/redact.go
```

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

---

## Architecture Decision Records

The following ADRs govern the design decisions behind the security controls described on this page:

- [ADR-004: Data Classification and Cross-Boundary Data Flows](adr/004-data-classification.md) — provides the full data classification taxonomy that underlies the redaction rules, telemetry sanitisation, and URL credential stripping documented here. The audit log redaction rules are derived from the `SECRET` data class.
- [ADR-003: Trust Boundaries and Component Trust Levels](adr/003-trust-boundaries.md) — documents the telemetry and crash report boundaries (Tier 4, external services) and the controls — command name sanitisation, hash fingerprinting — applied before data crosses those boundaries.
- [ADR-005: Canonicalization Ownership](adr/005-canonicalization-ownership.md) — documents the schema validation and `NaN`/`Infinity` rejection that prevents malformed values from reaching the signing path.
