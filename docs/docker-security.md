# Docker Security and Hardened Runtime Profile

This document describes the security model applied to the Glassbox Docker image
and compose configuration, and explains how to run and test the hardened profile.

## Security properties

### Non-root execution

The final image creates a system user and group named `glassbox` with fixed
UID/GID **10001**. The `USER` directive in the Dockerfile switches to this user
before the `ENTRYPOINT`, so the glassbox process never runs as root inside the
container.

Using a fixed UID/GID means bind-mounted volume permissions are predictable:

```bash
# On the host, create directories with matching ownership
mkdir -p ./cache ./sessions
chown 10001:10001 ./cache ./sessions
```

The `docker-compose.yml` file reinforces this with `user: "10001:10001"` on
every service so the setting cannot be accidentally overridden by an image
rebuild that forgets to set `USER`.

### Read-only filesystem

The root filesystem is mounted read-only. Only three writable paths exist at
runtime:

| Path | Type | Purpose |
|------|------|---------|
| `/app/cache` | Named volume | RPC response cache (persists across restarts) |
| `/app/sessions` | Named volume | Debug session state (persists across restarts) |
| `/tmp` | tmpfs | In-flight scratch space (lost on container stop) |

This prevents an attacker who achieves code execution from modifying binaries,
configuration, or libraries inside the container.

To enable the read-only filesystem in a plain `docker run` invocation:

```bash
docker run --rm \
  --read-only \
  --tmpfs /tmp:size=64m,mode=1777 \
  -v glassbox-cache:/app/cache \
  -v glassbox-sessions:/app/sessions \
  -v "$(pwd)/glassbox.example.toml:/app/glassbox.toml:ro" \
  glassbox:local --help
```

### Capability dropping

All Linux capabilities are dropped with `cap_drop: [ALL]`. Glassbox is a
debugging CLI — it requires no kernel capabilities to operate. Dropping them
removes the ability to bind privileged ports, load kernel modules, or bypass
DAC permissions even if the container is compromised.

### No privilege escalation

`no-new-privileges:true` (via `security_opt`) prevents the glassbox process or
any child it spawns from gaining additional privileges through `setuid` binaries
or `sudo`. This also prevents `seccomp` and `AppArmor` profiles from being
bypassed via `execve` of a privileged binary.

## Health checks

### Default health check

The image includes a `HEALTHCHECK` that runs `/app/glassbox --version`:

```dockerfile
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD ["/app/glassbox", "--version"]
```

This confirms:
- The binary exists and is executable.
- All shared libraries and dynamic loader paths are intact (none — the binary
  is statically linked, so this is always true for the default image).
- The container has not been corrupted by a failed partial write.

The container transitions to **unhealthy** if the check fails three consecutive
times. Docker and compose use this signal to restart the container (when
`restart: on-failure` or `restart: unless-stopped` is configured).

### Testing the unhealthy path

The `glassbox-degraded` service in `docker-compose.yml` overrides the
entrypoint with a command that always exits non-zero, causing the health check
to fail immediately. Use it in CI to verify that your orchestration layer
(compose, Kubernetes, ECS) correctly detects and handles unhealthy containers:

```bash
# Start only the degraded service
docker compose --profile test up glassbox-degraded

# In another terminal, poll until unhealthy
until [ "$(docker inspect --format='{{.State.Health.Status}}' glassbox-degraded)" = "unhealthy" ]; do
  sleep 2
done
echo "Container correctly reported unhealthy"
docker compose --profile test down
```

### Custom health check for daemon / metrics mode

When glassbox is used with a metrics endpoint (e.g. `--telemetry` with an
OTLP exporter), override the health check in your compose override file:

```yaml
# docker-compose.override.yml
services:
  glassbox:
    healthcheck:
      test: ["CMD", "wget", "-qO-", "http://localhost:2112/healthz"]
      interval: 15s
      timeout: 5s
      start_period: 30s
      retries: 3
```

## Compose profiles

| Profile | Purpose |
|---------|---------|
| *(default)* | Runs `glassbox --help` and exits. Safe for testing the image. |
| `debug` | Interactive TTY session for real debug runs. Passes `GLASSBOX_RPC_URL` and `GLASSBOX_NETWORK` from the host environment. |
| `test` | Starts `glassbox-degraded` (always unhealthy) for CI health-check testing. |
| `tracing` | Adds a Jaeger all-in-one container for distributed trace collection. |

```bash
# Default (safe, exits immediately)
docker compose up

# Interactive debug session
docker compose --profile debug run --rm glassbox-debug debug <tx-hash>

# Test unhealthy detection
docker compose --profile test up glassbox-degraded

# With Jaeger tracing
docker compose --profile tracing up
```

## Secret injection

Glassbox reads credentials exclusively from environment variables; it does not
consume Docker-native `secrets:` mounts. The two runtime credentials are:

| Secret | Environment variable | Used by |
|--------|---------------------|---------|
| RPC authentication token | `GLASSBOX_RPC_TOKEN` | All RPC-backed commands (`debug`, `explain`, `dry-run`, `shell`, `compare`, `profile`, `debug bundle create`) |
| Release manifest signing key | `GLASSBOX_MANIFEST_SIGNING_KEY` | CI release pipeline only — **never** pass this to a runtime container |

### Passing the RPC token to a container

Inject the token at runtime so it never touches the image layer or the
repository. The recommended pattern uses `docker compose run --env`:

```bash
# Read the token without echoing it, then pass it to the container.
read -rs GLASSBOX_RPC_TOKEN
export GLASSBOX_RPC_TOKEN

docker compose --profile debug run --rm \
  -e GLASSBOX_RPC_TOKEN \
  glassbox-debug debug --network testnet <tx-hash>
```

In CI, map the token from the platform secret store:

```yaml
# GitHub Actions example
- name: Run debug
  env:
    GLASSBOX_RPC_TOKEN: ${{ secrets.GLASSBOX_RPC_TOKEN }}
  run: |
    docker compose --profile debug run --rm \
      -e GLASSBOX_RPC_TOKEN \
      glassbox-debug debug --network testnet $TX_HASH
```

The `glassbox-debug` service in `docker-compose.yml` already declares
`GLASSBOX_RPC_TOKEN` as a pass-through variable (empty default), so the token
is forwarded when it is set in the calling environment without any further
compose file edits.

Confirm the token was picked up without printing it:

```bash
docker compose --profile debug run --rm \
  -e GLASSBOX_RPC_TOKEN \
  glassbox-debug config show --json | python3 -m json.tool | grep rpc_token
# Expected: "value": "[redacted]"
```

For the full token resolution order and additional safe-handling techniques,
see [RPC Bearer Token Examples](./rpc-token-examples.md).

### What is never baked into the image

- No `ENV` directive in the Dockerfile sets any credential.
- The `GLASSBOX_MANIFEST_SIGNING_KEY` variable is consumed only inside the CI
  `generate-release-manifest` tool and is never forwarded to a container.
- The `glassbox.example.toml` file (bind-mounted read-only as
  `/app/glassbox.toml`) contains no credentials. Any real `rpc_token` or
  `rpc_url` with embedded credentials belongs in an untracked local config
  file, not in `glassbox.example.toml`.

## Running the full hardened stack

```bash
# 1. Build the hardened image
docker compose build

# 2. Verify the image runs as non-root
docker compose run --rm glassbox id
# Expected: uid=10001(glassbox) gid=10001(glassbox)

# 3. Verify read-only filesystem (write attempt must fail)
docker compose run --rm glassbox sh -c "touch /app/test 2>&1 || echo 'read-only filesystem confirmed'"

# 4. Verify writable paths work
docker compose run --rm glassbox sh -c "echo ok > /tmp/probe && echo '/tmp is writable'"

# 5. Run a debug session
GLASSBOX_RPC_URL=https://soroban-testnet.stellar.org \
  docker compose --profile debug run --rm glassbox-debug \
  debug --network testnet <tx-hash>
```

## Verifying the image is not running as root in production

```bash
# Check the running process UID
docker exec <container-name> id

# Check effective capabilities (should be empty)
docker exec <container-name> cat /proc/1/status | grep Cap
# CapInh: 0000000000000000
# CapPrm: 0000000000000000
# CapEff: 0000000000000000

# Attempt a privileged operation (must fail)
docker exec <container-name> chown root /app/glassbox 2>&1
# chown: /app/glassbox: Read-only file system
```

## Image verification workflow

Every release image is built from the same source commit recorded in
`manifest.json`. To verify the image matches the release:

1. Pull the image digest from the release notes or OCI registry.
2. Check the `org.opencontainers.image.revision` label matches the
   `commit` field in `manifest.json`.
3. Optionally rebuild from source and compare layer digests.

```bash
# Inspect OCI labels
docker inspect glassbox:local \
  --format '{{json .Config.Labels}}' | python3 -m json.tool
```

## Security checklist

- [ ] Image runs as UID 10001 (non-root)
- [ ] `--read-only` flag is set (or `read_only: true` in compose)
- [ ] `/tmp` is a tmpfs mount, not a bind mount to the host
- [ ] `/app/cache` and `/app/sessions` are named volumes or explicit bind mounts
- [ ] `cap_drop: [ALL]` is set
- [ ] `no-new-privileges:true` is set
- [ ] Health check is configured and transitions to unhealthy on failure
- [ ] No secrets are baked into the image (inject via `-e`/`--env` at `docker compose run` time, or via CI secret stores)
- [ ] `GLASSBOX_MANIFEST_SIGNING_KEY` is never passed to a runtime container (CI pipeline only)
