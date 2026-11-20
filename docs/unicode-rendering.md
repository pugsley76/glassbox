# Unicode Rendering Modes

Glassbox renders status indicators and decorative symbols differently depending
on whether output is going to an interactive terminal or a plain-text consumer
(CI log, pipe, screen reader, `NO_COLOR` mode). This document describes the two
rendering modes, the symbol mapping, the emoji-removal tool, and how to verify
correct behavior.

---

## Two rendering modes

### TTY mode (interactive terminal)

Active when `internal/terminal.ANSIRenderer.IsTTY()` returns `true`. Requires:
- `FORCE_COLOR` is set, **or**
- `NO_COLOR` is **not** set, **and**
- `TERM` is not `dumb`, **and**
- `termctx.GlobalNonInteractive()` is `false`, **and**
- stdout is an actual TTY (detected via `go-isatty`).

In TTY mode, `Colorize` wraps text in ANSI SGR escape sequences and `Symbol`
returns descriptive bracket-style ASCII strings (e.g. `[OK]`, `[TOOL]`).

### Plain-text mode (non-TTY)

Active when `IsTTY()` returns `false`. All triggers that disable TTY mode are
listed in the precedence table in [docs/no-color.md](no-color.md).

In plain-text mode:
- `Colorize` returns the input text unchanged — no escape sequences.
- `Symbol` returns a minimal ASCII representation (see table below).
- `ClearLine` is a no-op.
- `Success()`, `Warning()`, and `Error()` return `[OK]`, `[!]`, and `[X]`
  respectively, with no ANSI codes.

Plain-text mode is the path used by CI pipelines, screen readers, `--no-color`,
`--non-interactive`, `NO_COLOR=1`, `GLASSBOX_NO_COLOR=1`, and `TERM=dumb`.

---

## Symbol name → output mapping

`Renderer.Symbol(name string) string` accepts the following logical names.
Both columns are pure ASCII (code points ≤ U+007E), so they are safe for every
terminal, log file, and screen reader.

| Name       | TTY output  | Plain-text (non-TTY) | Notes |
|------------|------------|----------------------|-------|
| `check`    | `[OK]`     | `[OK]`               | |
| `cross`    | `[FAIL]`   | `[X]`                | |
| `warn`     | `[!]`      | `[!]`                | |
| `arrow_r`  | `->`       | `->`                 | |
| `arrow_l`  | `<-`       | `<-`                 | |
| `target`   | `[TARGET]` | `>>`                 | |
| `pin`      | `*`        | `*`                  | |
| `wrench`   | `[TOOL]`   | `[*]`                | |
| `chart`    | `[STATS]`  | `[#]`                | |
| `list`     | `[LIST]`   | `[.]`                | |
| `play`     | `[PLAY]`   | `>`                  | |
| `book`     | `[DOC]`    | `[?]`                | |
| `wave`     | `[HELLO]`  | _(empty string)_     | Decorative only; intentionally silent in non-TTY. See note below. |
| `magnify`  | `[SEARCH]` | `[?]`                | |
| `logs`     | `[LOGS]`   | `[Logs]`             | |
| `events`   | `[NET]`    | `[Events]`           | |

**`wave` in plain-text mode:** `Symbol("wave")` returns an empty string when
`IsTTY()` is false. It is used only for decorative greetings and must produce
no output in CI or screen-reader contexts. Code that calls `Symbol("wave")` and
needs to produce visible output in all modes must use a different symbol name.
`TestSymbol_AllKnownNames` in `internal/terminal/platform_tty_test.go`
explicitly skips `wave` to document this contract.

Unknown symbol names are returned unchanged (forward-compatibility).

---

## Sentinel methods

The following `Renderer` methods always return a **non-empty** string regardless
of TTY state. This is enforced by `TestSuccessWarningError_NeverEmptyOnAnyPlatform`
in `internal/terminal/platform_tty_test.go`.

| Method      | TTY output             | Plain-text output |
|-------------|------------------------|-------------------|
| `Success()` | `\033[32m[OK]\033[0m`  | `[OK]`            |
| `Warning()` | `\033[33m[!]\033[0m`   | `[!]`             |
| `Error()`   | `\033[31m[X]\033[0m`   | `[X]`             |

---

## Emoji removal tool (`emojis.py`)

The repository enforces a no-emoji policy on source files, documentation, and
configuration. The `emojis.py` script at the repository root scans for
Unicode emoji and replaces them with the ASCII equivalents in the mapping table
above.

### Usage

```bash
# Check for emoji without modifying files (exits 1 if any are found)
python emojis.py --check

# Fix all found emoji in-place
python emojis.py --fix

# Scan a specific directory
python emojis.py --check docs/
```

### Scanned file types

`.md`, `.go`, `.ts`, `.js`, `.tsx`, `.jsx`, `.toml`, `.yml`, `.yaml`, `.json`

### Excluded directories

`.git`, `node_modules`, `vendor`, `dist`, `out`, `coverage`, `.next`, `.vscode`

### Duplicate-key rule

Each emoji may appear only once as a key in the `replacements` dict in
`emojis.py`. A duplicate key causes Python to silently discard the first
mapping, leaving one emoji untranslated. The rocket emoji (`🚀`) previously
appeared twice (`[START]` and `[DEPLOY]`); only `[START]` is retained after
deduplication. If a new replacement is needed for the same emoji, update the
existing entry rather than adding a second key.

---

## Running the tests

```bash
# ANSIRenderer color and symbol behavior
go test ./internal/terminal/... -run ANSI
go test ./internal/terminal/... -run Symbol
go test ./internal/terminal/... -run TTY

# Cross-platform TTY detection (NO_COLOR, FORCE_COLOR, TERM=dumb)
go test ./internal/terminal/... -run CrossPlatform

# Emoji scan (check mode — no file modifications)
python emojis.py --check .
```

---

## Related documents

- [no-color.md](no-color.md) — no-color mode triggers and precedence
- [command-development-guide.md](command-development-guide.md) §8 — accessibility rules for new commands
