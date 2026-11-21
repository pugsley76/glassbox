# Architecture Decision Records (ADRs)

This directory contains Architecture Decision Records (ADRs) for the Glassbox project. ADRs capture important architectural decisions and their rationale.

## ADR Format

Each ADR follows the standard format:
- **Status**: Accepted, Proposed, Deprecated, or Superseded
- **Context**: Problem statement and background
- **Decision**: The chosen solution with technical details
- **Rationale**: Why this decision was made over alternatives
- **Implementation**: How the decision is implemented
- **Consequences**: Effects of this decision

## ADR Index

Every row below links to a file that exists in this directory. When adding an
ADR, add a matching row so the index stays complete.

| ADR | Title | Status | Date |
|-----|-------|--------|------|
| [001](001-snapshot-caching.md) | Snapshot Caching Strategy using Bincode and SHA256 | Accepted | 2026-02-26 |
| [002](002-hsm-integration.md) | HSM Integration for Cryptographic Operations | Accepted | 2026-02-26 |
| [003](003-trust-boundaries.md) | Trust Boundaries and Component Trust Levels | Accepted | 2026-08-31 |
| [004](004-data-classification.md) | Data Classification and Cross-Boundary Data Flows | Accepted | 2026-08-31 |
| [005](005-canonicalization-ownership.md) | Canonicalization Ownership | Accepted | 2026-08-31 |
| [006](006-provider-isolation.md) | Provider Isolation | Accepted | 2026-08-31 |
| [007](007-offline-guarantees.md) | Offline Guarantees | Accepted | 2026-08-31 |
| [008](deeplink-parameters.md) | Deeplink Parameters | Accepted | 2026-08-31 |
| [009](simulation-failure-diagnostics.md) | Simulation Failure Diagnostics | Accepted | 2026-08-31 |

## ADR Process

### Proposing an ADR

1. Create a new ADR file using the next sequential number and the filename
   convention `NNN-kebab-case-title.md` (for example,
   `010-new-decision-title.md`)
2. Set status to "Proposed"
3. Add a row to the [ADR Index](#adr-index) using the exact table format:
   `| [NNN](NNN-kebab-case-title.md) | Title | Proposed | YYYY-MM-DD |`
4. Submit a pull request for review
5. Address feedback and revise as needed

### Accepting an ADR

1. ADR must be reviewed by at least one maintainer
2. Consensus should be reached among the team
3. Update status to "Accepted"
4. Implement the decision if not already done

### Modifying an ADR

1. If an ADR needs changes, create a new ADR that supersedes it
2. Update the original ADR's status to "Superseded"
3. Reference the new ADR in the original

## ADR Template

```markdown
# ADR-XXX: [Title]

## Status
[Proposed | Accepted | Deprecated | Superseded]

## Context
[Problem statement and background]

## Decision
[The chosen solution]

## Rationale
[Why this decision was made]

## Consequences
[Effects of implementing this decision]
```

## Adding an ADR

The filename convention is `NNN-kebab-case-title.md`, where `NNN` is the next
zero-padded sequential number. The index row must match the file exactly:

```markdown
| [010](010-new-decision-title.md) | New Decision Title | Proposed | 2026-09-01 |
```

### Validating the ADR Index

The index is checked by the documentation validator. Run the link check from
the repository root:

```bash
scripts/validate-docs.sh --links
```

- Exit code `0` means every link in the index resolves to an existing file.
- Exit code `1` means at least one link is broken.

To run the full documentation suite (links plus other checks):

```bash
scripts/validate-docs.sh --all
```

### Failure Example

If a row points at a file that does not exist, for example:

```markdown
| [010](010-missing-adr.md) | Missing ADR | Proposed | 2026-09-01 |
```

then `scripts/validate-docs.sh --links` reports a `Broken link` for
`docs/adr/010-missing-adr.md` and exits with code `1`. Fix the filename in the
row (or add the missing file) and re-run the validator until it exits `0`.

## Related Documentation

- [Architecture Overview](../architecture-dataflow.md)
- [Contributing Guidelines](../../README.md)
- [Technical Specifications](../schema/README.md)
