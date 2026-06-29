# No-Color Mode

Glassbox outputs ANSI escape sequences for colors and formatting by default.
This can interfere with log capture systems, CI pipelines, and terminal
emulators that do not support ANSI codes. No-color mode disables all ANSI
output and produces plain, machine-readable text.

## Enabling no-color mode

### CLI flag (recommended)

Pass `--no-color` as a persistent global flag before any subcommand:

```
glassbox --no-color debug <tx-hash>
glassbox --no-color trace --print <trace-file>
```

The flag is persistent, so it applies to every subcommand in the invocation.

### Environment variables

Two environment variables are honoured:

| Variable          | Description                                                  |
|-------------------|--------------------------------------------------------------|
| `NO_COLOR`        | Standard convention (<https://no-color.org>). Any non-empty value disables color. |
| `GLASSBOX_NO_COLOR` | Glassbox-specific override. Useful when `NO_COLOR` is already claimed by other tools. |

```bash
# Using the standard convention
NO_COLOR=1 glassbox debug <tx-hash>

# Using the Glassbox-specific variable
GLASSBOX_NO_COLOR=1 glassbox debug <tx-hash>
```

Setting either variable is equivalent to passing `--no-color` on the command
line. The `--no-color` flag additionally propagates `NO_COLOR=1` to any child
processes spawned during the command.

## Precedence

No-color sources are checked in this order (first match wins):

1. `--no-color` CLI flag
2. `GLASSBOX_NO_COLOR` environment variable
3. `NO_COLOR` environment variable
4. `FORCE_COLOR` environment variable (overrides automatic detection to *enable* color)
5. `TERM=dumb`
6. Automatic TTY detection (colors enabled only when stdout is a terminal)

## Affected output

No-color mode disables ANSI escape sequences across all Glassbox output:

- Execution trace trees (`glassbox trace --print`)
- Status indicators (`[OK]`, `[!]`, `[FAIL]`)
- Contract boundary separators
- All `Colorize`/`Success`/`Warning`/`Error` visualizer helpers

The structured JSON output produced by `--json` or `glassbox trace --output-json`
is unaffected — it never contains ANSI codes.

## CI / log capture usage

For CI pipelines, set `NO_COLOR` or `GLASSBOX_NO_COLOR` in the environment
rather than modifying every command invocation:

```yaml
# GitHub Actions example
env:
  GLASSBOX_NO_COLOR: "1"
```

```bash
# Shell script example
export NO_COLOR=1
glassbox debug "$TX_HASH" | tee debug.log
```
