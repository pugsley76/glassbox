# Deep Link Parameters and Protocol Registration

This guide covers the `glassbox://` URI scheme, the protocol registration commands, and how to verify protocol registration on Windows, macOS, and Linux.

For URI parameter semantics (query parameters, encoding, and supported actions), see [docs/adr/deeplink-parameters.md](adr/deeplink-parameters.md).

## Overview

Glassbox registers the custom URI scheme `glassbox://` with the operating system so that links such as `glassbox://open?path=...` are routed to the Glassbox application.

Protocol registration is managed through the following commands:

| Command | Purpose |
| --- | --- |
| `protocol:register` | Register the `glassbox://` scheme with the operating system. |
| `protocol:verify` | Verify that the scheme is registered and points at the current application. |
| `protocol:status` | Report the current registration state. |
| `protocol:diagnose` | Collect diagnostic details about the registration. |
| `protocol:repair` | Re-register the scheme to fix a broken or stale registration. |

Each platform stores registration in a different location. The sections below describe the expected location, how to verify it, and how to troubleshoot a missing registration.

## Windows

On Windows, protocol registration is stored in the registry under the `erst` key:

- Registry key: `HKCU\Software\Classes\erst`
- The key must contain a `URL Protocol` value.
- The command used to open the scheme is stored under `shell\open\command`.

### Verify

Run the verification script from the repository root:

```powershell
powershell -ExecutionPolicy Bypass -File scripts/verify_protocol_registration_windows.ps1
```

Expected output on success:

```
Windows protocol registration verified
```

### Failure example

If the registry key is missing, the script reports the missing key and exits non-zero. For example:

```
Windows protocol registration failed: missing registry key HKCU\Software\Classes\erst
```

To fix a missing registration, run `protocol:register` (or `protocol:repair` if a stale registration exists), then re-run the verification script.

## macOS

On macOS, protocol registration is declared in the application bundle's `Info.plist` under the `CFBundleURLSchemes` key.

- App bundle path: the installed `Glassbox.app` bundle.
- `Info.plist` key: `CFBundleURLSchemes` must include `glassbox`.

### Verify

Run the verification script from the repository root:

```bash
bash scripts/verify_protocol_registration_macos.sh
```

Expected output on success:

```
macOS protocol registration verified
```

### Failure example

If the `Info.plist` is missing or does not declare the scheme, the script reports the missing plist entry and exits non-zero. For example:

```
macOS protocol registration failed: missing CFBundleURLSchemes entry in Info.plist
```

To fix a missing registration, run `protocol:register` (or `protocol:repair`), then re-run the verification script.

## Linux

On Linux, protocol registration is stored in a desktop entry file.

- Desktop file path: `glassbox-protocol.desktop`
- The desktop file must declare `MimeType=x-scheme-handler/Glassbox`.

### Verify

Query the default handler for the scheme:

```bash
xdg-mime query default x-scheme-handler/Glassbox
```

Expected output on success:

```
glassbox-protocol.desktop
```

You can also run the verification script from the repository root:

```bash
bash scripts/verify_protocol_registration_linux.sh
```

Expected output on success:

```
Linux protocol registration verified
```

### Troubleshooting a missing handler

If `xdg-mime query default x-scheme-handler/Glassbox` returns no output, the scheme has no registered handler. Check that the desktop file exists and declares the `MimeType=x-scheme-handler/Glassbox` entry, then re-register with `protocol:register` (or `protocol:repair`) and re-run the verification script.

## Running the integration test

The protocol registration integration test is opt-in. Enable it with the following environment variables:

- `GLASSBOX_RUN_PROTOCOL_REGISTRATION_TESTS=1` — enables the integration test.
- `GLASSBOX_PROTOCOL_VERIFY_SCRIPT` — path to the platform verification script to run.

Example (Linux):

```bash
GLASSBOX_RUN_PROTOCOL_REGISTRATION_TESTS=1 \
GLASSBOX_PROTOCOL_VERIFY_SCRIPT=scripts/verify_protocol_registration_linux.sh \
go test ./...
```

Replace the script path with the platform-appropriate script (`scripts/verify_protocol_registration_windows.ps1` or `scripts/verify_protocol_registration_macos.sh`).

## See also

- [docs/adr/deeplink-parameters.md](adr/deeplink-parameters.md) — URI parameter semantics.