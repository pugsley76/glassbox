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

Three environment variables affect color output:

| Variable      | Effect | Precedence |
|---------------|--------|------------|
| `NO_COLOR`    | Any value present (even empty) disables color. Standard convention (<https://no-color.org>). | Highest — overrides `FORCE_COLOR` |
| `FORCE_COLOR` | Any non-empty value forces color on, even when stdout is not a TTY. | Overrides automatic TTY detection |
| `TERM=dumb`   | Disables color; used by headless CI environments. | Below `FORCE_COLOR` |

> **Note:** `GLASSBOX_NO_COLOR` is documented in some places as a Glassbox-specific
> override. In practice, the `internal/terminal.ANSIRenderer` only checks the
> standard `NO_COLOR` variable. The `--no-color` CLI flag sets `NO_COLOR` in
> the process environment, so child processes honour it automatically.

```bash
# Standard convention (recommended)
NO_COLOR=1 glassbox debug <tx-hash>

# Force color in a CI environment that sets TERM=dumb
FORCE_COLOR=1 glassbox debug <tx-hash>
```

**Note:** The `--non-interactive` flag (and CI/pipe detection) also suppresses
color output independently of `NO_COLOR`. See `docs/command-development-guide.md`
for details on non-interactive mode.

**Note:** `GLASSBOX_NO_COLOR` is resolved in `PersistentPreRunE` in
`internal/cmd/root.go`, which sets `NO_COLOR=1` and calls
`visualizer.SetNoColor(true)`. The `internal/terminal.ANSIRenderer` reads
only `NO_COLOR` directly (not `GLASSBOX_NO_COLOR`), so the two-step path
through the CLI layer is required for `GLASSBOX_NO_COLOR` to take effect.
When using `ANSIRenderer` directly (e.g. in tests), set `NO_COLOR` instead.

## Precedence

Color detection is performed by `internal/terminal.ANSIRenderer.IsTTY()`.
Sources are checked in this order (first match wins):

1. `--non-interactive` flag (or `termctx.GlobalNonInteractive()`) — always disables color and cursor-control sequences
2. `NO_COLOR` environment variable (any value present) — disables color
3. `FORCE_COLOR` environment variable (non-empty) — forces color on
4. `TERM=dumb` — disables color
5. Automatic TTY detection: `isatty.IsTerminal(os.Stdout.Fd())` — colors enabled only when stdout is a terminal

**`NO_COLOR` always takes precedence over `FORCE_COLOR`.** If both are set,
color is disabled. This matches the [no-color.org](https://no-color.org) specification
and is verified by `TestTTYDetection_CrossPlatformEnvVars` in
`internal/terminal/platform_tty_test.go`.

Non-interactive mode suppresses all terminal control sequences including
cursor movement (`ClearLine`), regardless of the variables above.

## Affected output

No-color mode disables ANSI escape sequences across all Glassbox output:

- Execution trace trees (`glassbox trace --print`)
- Status indicators (`[OK]`, `[!]`, `[FAIL]`)
- Contract boundary separators
- All `Colorize`/`Success`/`Warning`/`Error`/`Symbol` calls on the `Renderer` interface

The structured JSON output produced by `--json` or `glassbox trace --output-json`
is unaffected — it never contains ANSI codes. `Colorize` returns plain text
when `IsTTY()` is false, so JSON consumers will never receive escape sequences.

### Symbol divergence between TTY and non-TTY modes

The `Symbol("cross")` call returns `"[FAIL]"` in TTY mode and `"[X]"` in
non-TTY mode. This is intentional: `"[FAIL]"` is more descriptive in a
terminal context, while `"[X]"` is the compact non-TTY form used in logs and
plain-text output. All other symbol names return the same string in both modes.

## CI / log capture usage

For CI pipelines, set `NO_COLOR` in the environment rather than modifying
every command invocation:

```yaml
# GitHub Actions example
env:
  NO_COLOR: "1"
```

```bash
# Shell script example
export NO_COLOR=1
glassbox debug "$TX_HASH" | tee debug.log
```

## Verification

### Automated verification

The repository includes a verification script to confirm no ANSI escape sequences are emitted when output is piped:

```bash
# Run the verification script
./scripts/verify_no_ansi.sh
```

The script builds the binary if needed and tests both piped output and `NO_COLOR=1` behavior.

### Unit tests

Key test files for no-color behavior:
- `internal/terminal/ansi_test.go` — `NO_COLOR`, `FORCE_COLOR`, `TERM=dumb` behavior; `NO_COLOR` overrides `FORCE_COLOR`
- `internal/terminal/platform_tty_test.go` — cross-platform env-var precedence; `NO_COLOR` overrides `FORCE_COLOR`; non-TTY `Colorize` strips ANSI
- `internal/cmd/no_color_test.go` — CLI flag and environment variable activation
- `internal/visualizer/color_test.go` — Visualizer color enabling logic
- `internal/termctx/termctx_test.go` — Terminal context and non-interactive mode

Run the no-color specific tests:

```bash
go test ./internal/cmd/... -run NoColor
go test ./internal/terminal/... -run ANSI
go test ./internal/terminal/... -run TTY
go test ./internal/terminal/... -run Colorize
```
