# Help snapshot golden files

This directory contains golden help-text snapshots used by `help_snapshot_test.go`.

## Generating snapshots

Run the following command from the repository root to populate or regenerate
all snapshots:

```bash
go test ./internal/cmd -run TestHelpSnapshot -update-help
```

## When to regenerate

Regenerate whenever intentional changes are made to any command's `Use`,
`Short`, `Long`, or `Example` fields. The test will fail with a diff and
instruct you to run the command above.

## What is checked

Each `TestHelpSnapshot_<Command>` test captures the output of
`<command> --help` and compares it byte-for-byte against the corresponding
`<command>.txt` file in this directory.

Covered commands (one `.txt` file each):

- `root.txt` — root command help
- `debug.txt` — `glassbox debug --help`
- `version.txt` — `glassbox version --help`
- `audit_sign.txt` — `glassbox audit:sign --help`
- `audit_verify.txt` — `glassbox audit:verify --help`
- `session.txt` — `glassbox session --help`
- `completion.txt` — `glassbox completion --help`
- `diagnostics.txt` — `glassbox diagnostics --help`
- `doctor.txt` — `glassbox doctor --help`
