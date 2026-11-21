# audit:verify example assets

Synthetic, deterministic inputs for the walkthrough in
[`docs/audit-verify-command.md`](../../audit-verify-command.md#end-to-end-local-workflow).

- `payload.json` — unsigned payload with the fixed timestamp `2026-01-01T00:00:00.000Z`
- `payload-schema.json` — schema accepted by `audit:verify --schema`

No key material is stored here. The walkthrough generates throwaway test keys
in a temporary directory with `glassbox audit:keygen`.

## Command reference

`glassbox audit:verify` accepts the following flags:

| Flag | Purpose |
| --- | --- |
| `--audit-log <path>` | Path to the signed audit log to verify (required). |
| `--public-key <path>` | Public key used to verify the log signature. |
| `--schema <path>` | JSON schema the payload must satisfy. |
| `--json` | Emit machine-readable JSON instead of human-readable text. |
| `--previous-signature-hash <hex>` | Expected hash of the previous log entry (chain continuity). |
| `--trust-roots <path>` | PEM bundle of trusted roots for certificate validation. |
| `--allowed-issuers <list>` | Comma-separated list of allowed certificate issuers. |
| `--check-validity` | Enforce certificate validity windows. |
| `--revoked-certs <path>` | Path to a revocation list. |
| `--revocation-mode <mode>` | Revocation checking mode (e.g. `enforce`, `warn`). |
| `--policy-config <path>` | Policy configuration file applied during verification. |

## Success path

Given a signed log produced by the walkthrough:

```sh
glassbox audit:verify --audit-log signed-audit.json
```

Expected exit code: `0`.

Sample output:

```text
audit log verified: signed-audit.json
```

## Failure path

If the payload has been tampered with (or does not match the supplied
`--schema`), verification fails with a non-zero exit code:

```sh
glassbox audit:verify --audit-log signed-audit.json --schema payload-schema.json
```

Expected exit code: non-zero (e.g. `1`).

Sample error output:

```text
error: audit log verification failed: payload does not match schema
```

Use `--json` to receive the same result as a structured object when scripting
against the command.