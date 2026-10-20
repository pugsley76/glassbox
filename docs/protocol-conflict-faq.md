# Protocol Scheme Conflict FAQ

This page explains what to do when another application claims the `glassbox://` URI scheme, so that `glassbox://` links no longer open Glassbox. It describes the symptoms, a safe read-only inspection order, and the state-changing repair step.

> **Important:** Do not edit registry data, desktop files, or other OS registration data by hand. Use only the supported `glassbox protocol:*` commands described below. Manual edits can leave the registration in an inconsistent state that the supported commands cannot detect or repair.

## Symptoms

You may be affected by a scheme conflict if any of the following occur:

- `glassbox protocol:diagnose` reports a conflict (another application is registered for `glassbox://`).
- `glassbox protocol:verify` fails.
- The operating system launches a different application when you open a `glassbox://` link.
- `glassbox://` links do nothing, or open the wrong application, after installing or updating another program.

A conflict usually happens when another application registers the same URI scheme, either during its installation or through a user setting. Glassbox does not remove or override another application's registration silently.

## Safe inspection order (read-only)

Run these commands in order. All of them are read-only and do not change the registration.

1. **Check current status.**

   ```
   glassbox protocol:status
   ```

   This reports whether Glassbox is currently registered for `glassbox://` and which handler the OS will use.

2. **Diagnose the conflict.**

   ```
   glassbox protocol:diagnose
   ```

   This inspects the registration and reports any conflict, including which other application owns the scheme. For machine-readable output, add the JSON format flag:

   ```
   glassbox protocol:diagnose --format json
   ```

3. **Verify the registration.**

   ```
   glassbox protocol:verify
   ```

   This confirms that the registration is valid and that `glassbox://` links resolve to Glassbox.

On Linux and macOS you can also run the platform verification scripts, which are read-only checks:

- `scripts/verify_protocol_registration_linux.sh`
- `scripts/verify_protocol_registration_macos.sh`

These scripts inspect the current registration and report problems without modifying anything.

## Repairing the conflict

> **State-changing step:** `glassbox protocol:repair` modifies registration state. It overwrites the existing `glassbox://` registration so that Glassbox becomes the handler. Run it only after you have completed the inspection above and understand which application currently owns the scheme.

```
glassbox protocol:repair
```

After repairing, re-run the read-only checks to confirm the result:

```
glassbox protocol:status
glassbox protocol:verify
```

### Other state-changing commands

The following commands also modify registration state and should be used deliberately:

- `glassbox protocol:register` — registers Glassbox as the `glassbox://` handler.
- `glassbox protocol:unregister` — removes the Glassbox registration for `glassbox://`.

## If the conflicting application is still needed

If the other application that owns `glassbox://` is still installed and you rely on it, do not run `glassbox protocol:repair` blindly. Repairing will take over the scheme and may break that application's links.

Instead:

1. Uninstall the conflicting application, or reconfigure it so it no longer registers `glassbox://`.
2. Re-run the read-only checks (`glassbox protocol:status`, `glassbox protocol:diagnose`, `glassbox protocol:verify`) to confirm the conflict is gone.
3. If Glassbox is still not the handler, run the state-changing repair step:

   ```
   glassbox protocol:repair
   ```

## Related commands

- [`glassbox protocol:status`](#safe-inspection-order-read-only) — read-only status check.
- [`glassbox protocol:diagnose`](#safe-inspection-order-read-only) — read-only conflict diagnosis (supports `--format json`).
- [`glassbox protocol:verify`](#safe-inspection-order-read-only) — read-only verification.
- [`glassbox protocol:repair`](#repairing-the-conflict) — state-changing repair that overwrites the registration.
- [`glassbox protocol:register`](#other-state-changing-commands) — state-changing registration.
- [`glassbox protocol:unregister`](#other-state-changing-commands) — state-changing removal of the registration.