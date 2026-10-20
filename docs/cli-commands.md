# CLI Commands: Cancellation & Interruption

This page describes what happens when you interrupt a long-running Glassbox
command with `Ctrl+C` (`SIGINT`) or `SIGTERM`, which commands implement
graceful shutdown, and what happens to partially written artifacts.

For per-command flags, see the individual command references such as the
[debug command](./debug-command.md), [watch mode](./WATCH_MODE.md), and
[trace profiling](./trace-profiling.md).

---

## How interruption works

Signal handling is installed once, in the root command, for every Glassbox
subcommand:

1. **First `Ctrl+C` / `SIGTERM`** — the command's context is cancelled and the
   registered shutdown hooks run in reverse registration order (LIFO), bounded
   by a **3-second** total timeout. Glassbox prints
   `Interrupted. Shutting down...` and exits with code **130**.
2. **Second `Ctrl+C` / `SIGTERM`** — received while hooks are still running,
   it forces an immediate exit with code **130**. Hooks that have not finished
   are abandoned.

Exit code 130 is reserved for interruption; see [exit codes](./exit-codes.md).
A cancelled run never reports success, even if some output was written before
the signal arrived.

> Cancellation only takes effect early in commands that observe the cancelled
> context. A command that does not observe it keeps running until its current
> work completes and then exits with code 130. See the support matrix below.

---

## Support matrix

| Command | Observes cancellation | Shutdown hooks registered | Notes |
|---------|-----------------------|---------------------------|-------|
| `glassbox debug` | Yes | `debug-simulator` (closes the simulator runner) | Stops between phases; see partial artifacts below |
| `glassbox debug --watch-files` | Yes | `debug-simulator` | Exits the watch loop on the next wait; no rerun is started |
| `glassbox debug --watch` | Yes | `debug-simulator` | Stops polling for the transaction; no debug run is performed |
| `glassbox dry-run` | Yes | `rpc-cache-flush`, `dry-run-simulator-runner` | RPC cache is flushed before exit |
| `glassbox upgrade` | Yes | `rpc-cache-flush`, `upgrade-simulator-runner` | RPC cache is flushed before exit |
| `glassbox validate` | **No** | None | Checks run to completion; see below |

Commands not listed here receive the same root signal handling (exit code 130
on interrupt) but do not register dedicated shutdown hooks. Do not rely on them
to clean up in-progress work.

---

## Partial artifact behavior

### `glassbox debug`

| Artifact | Flag | Behavior on interrupt |
|----------|------|-----------------------|
| Trace export | `--trace-output`, `--generate-trace` | Export is skipped if the context is already cancelled (`Trace export skipped: operation was cancelled.`). If cancellation interrupts an in-progress export, the partial file is removed (`Trace export cancelled — partial file removed: <path>`). |
| Snapshot registry | `--save-snapshots` | Written to `<path>.tmp` and then renamed into place, so `<path>` is either the previous file or a complete new one, never a truncated one. A stray `<path>.tmp` may remain only if the process is force-killed during the write. |
| Flamegraph | `--profile` | Written in a single write after simulation. If the run is interrupted before that point, no flamegraph file is produced. |
| Call graph SVG | `--export-svg` | Written in a single write after simulation; not produced if the run is interrupted first. |

The simulator runner is closed by the `debug-simulator` hook, so no simulator
child process is left running after a clean (single-signal) interrupt.

### `glassbox debug --watch-files`

Each rerun is a full debug session, so the table above applies to the run that
was in progress. Artifacts from earlier, completed reruns are left untouched.
Once interrupted, the watcher stops and no further reruns are scheduled. See
[watch mode troubleshooting](./WATCH_MODE.md#troubleshooting-watch-mode).

### `glassbox validate`

`validate` does not observe cancellation and registers no shutdown hooks:

- The first `Ctrl+C` is acknowledged, but the running checks continue until
  they finish. The command then exits with code 130 instead of 0, 1, or 2.
- Press `Ctrl+C` a second time to force an immediate exit.
- The `--output` report is written once, after all checks complete. If you
  force-exit before that, no report is written and any existing file at that
  path is left as it was.

Treat a 130 exit from `validate` as "no result"; rerun it to get a valid
report.

---

## Interrupting from scripts

Send the signal only to the Glassbox process you started. Do not use
`pkill glassbox` or `killall`, which also terminate unrelated Glassbox sessions
(for example, a `glassbox daemon` or a teammate's watcher on a shared host).

```bash
# Start a bounded watch session in the background and remember its PID
glassbox debug <transaction-hash> --watch-files &
GLASSBOX_PID=$!

# ... later: request a graceful shutdown of that process only
kill -INT "$GLASSBOX_PID"
wait "$GLASSBOX_PID"
echo "exit code: $?"   # 130 after an interrupt
```

To bound a run in CI without killing other processes, use `timeout`, which
signals only the child it started:

```bash
# Sends SIGTERM after 5 minutes, then SIGKILL 10 seconds later if still running
timeout --signal=TERM --kill-after=10s 5m \
  glassbox debug <transaction-hash> --trace-output ./trace.json
```

Prefer `TERM` or `INT` over `KILL`. `SIGKILL` cannot be caught, so shutdown
hooks do not run and `.tmp` files may be left behind.

---

## See also

- [Debug command reference](./debug-command.md)
- [Watch mode](./WATCH_MODE.md)
- [Exit codes](./exit-codes.md)
- [Trace export validation](./trace-export-validation.md)
- [Session locking](./session-locking.md)
