# Environment Variables

This document provides a comprehensive reference for all `GLASSBOX_*` environment variables used by Glassbox.

For stability levels and deprecation policies, see [Compatibility Matrix](compatibility-matrix.md).

---

## Configuration Precedence

Environment variables follow the configuration precedence order:
1. CLI flags (highest priority)
2. Environment variables
3. Config files
4. Built-in defaults (lowest priority)

See [Config Show Command](config-show.md#configuration-precedence) for details.

---

## Network and RPC

| Variable | Description | Stability | Default |
|----------|-------------|-----------|---------|
| `GLASSBOX_NETWORK` | Stellar network: `testnet`, `mainnet`, `futurenet`, `standalone`, `public` | stable | `testnet` |
| `GLASSBOX_RPC_URL` | Custom RPC endpoint URL | stable | network-specific default |
| `GLASSBOX_RPC_URLS` | Comma-separated list of RPC URLs for failover | stable | (none) |
| `GLASSBOX_SOROBAN_RPC_URLS` | Comma-separated list of Soroban RPC URLs for adaptive failover | stable | (none) |
| `GLASSBOX_RPC_TOKEN` | RPC authentication token | stable | (none) |
| `GLASSBOX_FAILOVER_STRATEGY` | RPC failover strategy: `weighted`, `sticky`, `round_robin` | stable | `weighted` |
| `GLASSBOX_FAILURE_THRESHOLD` | Number of failures before circuit breaker opens | stable | `5` |
| `GLASSBOX_RETRY_TIMEOUT` | Seconds to wait before retrying a failed endpoint | stable | `60` |
| `GLASSBOX_REQUEST_TIMEOUT` | HTTP request timeout in seconds for all RPC calls | stable | `15` |

---

## Simulator and Build

| Variable | Description | Stability | Default |
|----------|-------------|-----------|---------|
| `GLASSBOX_SIMULATOR_PATH` | Path to the Glassbox simulator binary | stable | (auto-detected) |
| `GLASSBOX_BUILD_MANIFEST` | Path to `glassbox-build-manifest.json` for cross-machine source mapping | stable | (auto-discovery) |

---

## Logging and Debugging

| Variable | Description | Stability | Default |
|----------|-------------|-----------|---------|
| `GLASSBOX_LOG_LEVEL` | Log level: `trace`, `debug`, `info`, `warn`, `error` | stable | `info` |
| `GLASSBOX_LOG_FORMAT` | Log format: `json` or text | stable | text |

---

## Caching and Performance

| Variable | Description | Stability | Default |
|----------|-------------|-----------|---------|
| `GLASSBOX_CACHE_PATH` | Directory for source map cache | stable | `~/.Glassbox/cache` |
| `GLASSBOX_MAX_CACHE_SIZE` | Maximum cache size in bytes (supports `KB`, `MB`, `GB` suffixes) | stable | `0` (unlimited) |
| `GLASSBOX_MAX_TRACE_DEPTH` | Maximum depth of the call tree before truncation | stable | `50` |

---

## Telemetry

| Variable | Description | Stability | Default |
|----------|-------------|-----------|---------|
| `GLASSBOX_TELEMETRY` | Legacy boolean telemetry enable (use `GLASSBOX_TELEMETRY_LEVEL` instead) | stable | `false` |
| `GLASSBOX_TELEMETRY_LEVEL` | Telemetry level: `full`, `anonymous`, `disabled` | stable | `disabled` |
| `GLASSBOX_TELEMETRY_ENABLED` | Enable OpenTelemetry (distinct from legacy telemetry) | stable | `false` |
| `GLASSBOX_TELEMETRY_ENDPOINT` | OTLP exporter URL for OpenTelemetry | stable | (none) |
| `GLASSBOX_TELEMETRY_SAMPLE_RATE` | Fraction of high-frequency trace events to emit [0.0, 1.0] | stable | `1.0` |
| `GLASSBOX_TELEMETRY_ANONYMIZED` | Anonymize telemetry data (omit environment metadata) | stable | `true` |

**Note**: `GLASSBOX_TELEMETRY_LEVEL` takes precedence over `GLASSBOX_TELEMETRY` when both are set. See [Telemetry Metadata](telemetry-metadata.md) for details.

---

## Crash Reporting

| Variable | Description | Stability | Default |
|----------|-------------|-----------|---------|
| `GLASSBOX_CRASH_REPORTING` | Enable opt-in anonymous crash reporting | stable | `false` |
| `GLASSBOX_CRASH_ENDPOINT` | Custom HTTPS URL for crash reports | stable | (none) |
| `GLASSBOX_SENTRY_DSN` | Sentry Data Source Name for crash reporting | stable | (none) |

---

## Source Mapping

| Variable | Description | Stability | Default |
|----------|-------------|-----------|---------|
| `GLASSBOX_SKIP_SOURCE_MAPPING` | Disable DWARF source mapping (truthy values: `1`, `true`, `yes`) | stable | (none) |
| `GLASSBOX_SOURCE_MAP_CACHE` | Custom directory for source map cache | stable | (auto-detected) |
| `GLASSBOX_EXTERNAL_SOURCE_MAP` | JSON array of `{prefix, remote_url, branch}` for external source repos | stable | (none) |

---

## Session and Viewer

| Variable | Description | Stability | Default |
|----------|-------------|-----------|---------|
| `GLASSBOX_SESSION_DIR` | Directory for session files | stable | OS-specific user-data directory |
| `GLASSBOX_VIEWER_STATE_DIR` | Directory for viewer state sidecars | stable | `~/.Glassbox/viewer_state` |

---

## UI and Theming

| Variable | Description | Stability | Default |
|----------|-------------|-----------|---------|
| `GLASSBOX_THEME` | Color theme: `dark`, `light`, `none`, `default`, `deuteranopia`, `protanopia`, `tritanopia`, `high-contrast` | stable | (auto-detect) |
| `GLASSBOX_NO_COLOR` | Disable ANSI color output | stable | (none) |
| `NO_COLOR` | Standard convention to disable color (also supported) | stable | (none) |

---

## System Behavior

| Variable | Description | Stability | Default |
|----------|-------------|-----------|---------|
| `GLASSBOX_NON_INTERACTIVE` | Force non-interactive mode | stable | (none) |
| `GLASSBOX_NO_UPDATE_CHECK` | Disable update checker | stable | (none) |

---

## Audit Signing

| Variable | Description | Stability | Default |
|----------|-------------|-----------|---------|
| `GLASSBOX_AUDIT_PRIVATE_KEY_PEM` | PEM-encoded Ed25519 private key for audit signing | stable | (none) |
| `GLASSBOX_SOFTWARE_PRIVATE_KEY_HEX` | Hex-encoded Ed25519 private key for audit signing | stable | (none) |
| `GLASSBOX_SIGNING_PROVIDER` | Signing provider: `software`, `pkcs11` | stable | `software` |
| `GLASSBOX_SIGNER_TYPE` | Legacy signer type (deprecated in 0.3, use `GLASSBOX_SIGNING_PROVIDER`) | deprecated | (none) |
| `GLASSBOX_PKCS11_MODULE` | Path to PKCS#11 module (`.so`/`.dylib`/`.dll`) | stable | (none) |
| `GLASSBOX_PKCS11_PIN` | PKCS#11 user PIN | stable | (none) |
| `GLASSBOX_PKCS11_SLOT` | PKCS#11 slot index (non-negative integer) | stable | `0` |
| `GLASSBOX_PKCS11_TOKEN_LABEL` | PKCS#11 token label | stable | (none) |
| `GLASSBOX_PKCS11_KEY_LABEL` | PKCS#11 key label | stable | (none) |
| `GLASSBOX_PKCS11_KEY_ID` | PKCS#11 key ID (hex) | stable | (none) |
| `GLASSBOX_PKCS11_PUBLIC_KEY_PEM` | SPKI PEM public key for audit log embedding | stable | (none) |

---

## Plugin System

| Variable | Description | Stability | Default |
|----------|-------------|-----------|---------|
| `GLASSBOX_PLUGIN_ALLOW_UNTRUSTED` | Allow community and untrusted plugins to load | stable | `false` |

---

## Release and Signing

| Variable | Description | Stability | Default |
|----------|-------------|-----------|---------|
| `GLASSBOX_MANIFEST_SIGNING_KEY` | PEM private key for release manifest signing | stable | (none) |

---

## Internal/Test-Only Variables

The following variables are used internally for testing or development and should not be relied upon in production:

- `GLASSBOX_BINARY` / `GLASSBOX_BIN` - Test binary path
- `GLASSBOX_CONFIG_PASSPHRASE` - Test config encryption
- `GLASSBOX_CONTRACT_SEARCH_MAX_PAGES` - Test pagination limit
- `GLASSBOX_KMS_*` - AWS KMS test parameters
- `GLASSBOX_LANG` - Test locale
- `GLASSBOX_PROTOCOL_VERIFY_SCRIPT` - Test protocol verification
- `GLASSBOX_RUN_INTEGRATION_TESTS` - Test suite control
- `GLASSBOX_RUN_PROTOCOL_REGISTRATION_TESTS` - Test suite control
- `GLASSBOX_SANDBOX_NATIVE_TOKEN_CAP_STROOPS` - Sandbox test parameter
- `GLASSBOX_SESSION_ENCRYPTION` - Test session encryption
- `GLASSBOX_SESSION_KEY` / `GLASSBOX_SESSION_KEY_PASSPHRASE` - Test session keys
- `GLASSBOX_SESSION_KEY_PROVIDER` - Test session key provider
- `GLASSBOX_SIGN_KEY` - Test signing key
- `GLASSBOX_SIM_*` - Simulator test parameters
- `GLASSBOX_TEST_HORIZON_URL` - Test Horizon endpoint
- `GLASSBOX_VERSION` - Version override for telemetry

---

## Related Documentation

- [Configuration Precedence](config-show.md#configuration-precedence)
- [Compatibility Matrix](compatibility-matrix.md)
- [Telemetry Documentation](telemetry-metadata.md)
- [Source Mapping](source-mapping.md)
- [Audit Signing](audit-signing.md)
