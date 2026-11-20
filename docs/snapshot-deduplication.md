# Snapshot Storage with Content-Addressed Deduplication

## Overview

Snapshot storage in Glassbox now supports **content-addressed storage with automatic deduplication**. This means identical session snapshots are stored only once on disk, significantly reducing storage overhead and improving performance.

## Architecture

### Storage Layout

Snapshots are organized in a content-addressed store using stable hashing:

```
~/.glassbox/cache/
  snapshots/
    12345678_fedcba9876543210fedcba9876543210fedcba9876543210fedcba98765432.snap.json
    87654321_0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef.snap.json
    index.json
```

Each snapshot file is named as `<hash8>_<hash56>.snap.json` where:
- `<hash8>`: First 8 hex characters of SHA-256 content hash (for shell readability)
- `<hash56>`: Remaining 56 hex characters (complete 64-hex SHA-256 digest)
- `index.json`: Central index tracking all stored hashes for quick lookup

### Content Hashing

The content hash is a deterministic SHA-256 digest of the snapshot content (ledger entries and linear memory only). Two snapshots with identical:
- Ledger state (all key-value pairs)
- Linear memory

will produce the **same content hash**, ensuring reliable deduplication.

**What IS included in the hash**:
- Ledger entry keys and values
- Linear memory content
- Snapshot fingerprint (derived from ledger state)

**What's NOT included in the hash** (and therefore doesn't prevent deduplication):
- Transaction hash (different transactions can have identical snapshots)
- Network (testnet vs mainnet, etc.)
- `SavedAt` timestamp
- `SchemaVersion`
- Envelope XDR or Result Meta XDR

This design enables powerful deduplication: if Transaction A on testnet produces the same snapshot as Transaction B on mainnet, they'll share the same stored file, maximizing space savings.

## Key Features

### 1. Automatic Deduplication

When saving a snapshot:

```go
store := NewDedupStore(baseDir)
hash, isNew, err := store.SaveWithDedup(metadata, snapshot)

if isNew {
    // First time seeing this content - new file created
    // File stored at: baseDir/snapshots/<hash8>_<hash56>.snap.json
} else {
    // Identical snapshot already exists - no file written
    // Returned hash points to existing file
}
```

Deduplication is automatic *once a caller uses `SaveWithDedup`*: the store checks
the content hash and skips the write when the file already exists. The CLI
`glassbox snapshot save` uses `SavePersisted` directly and does not deduplicate.

### 2. Reliable Recovery

The storage is designed for reliable recovery from corruption:

```go
// Rebuild index from disk after corruption
recovered, err := store.RecoverFromCorruption()
// Returns number of valid snapshots recovered
```

This scans all `.snap.json` files and rebuilds the index from files that
`LoadPersisted` accepts (parseable JSON plus valid metadata/schema/network).
Fingerprint drift is **not** checked here; use `LoadWithDiagnostics()` or
`ValidateSnapshotBeforeReplay()` for that.

### 3. Content-Based Lookup

Load any snapshot directly by its content hash:

```go
ps, err := store.LoadByHash(hash)
// Returns: *PersistedSnapshot with full metadata and ledger state
```

Multiple transactions can produce identical snapshots. All refer to the same file:

```
Transaction A -> Network call -> Content Hash: abc123 -> File
Transaction B -> Network call -> Content Hash: abc123 -> Same File (deduplicated)
Transaction C -> Network call -> Content Hash: def456 -> Different File
```

### 4. Deduplication Statistics

Query deduplication statistics for monitoring and diagnostics:

```go
// Get count of unique snapshots
uniqueCount, err := store.QueryDedup()

// Get disk usage
bytes, err := store.GetDiskUsage()

// Find duplicate hashes (same content stored under multiple names)
duplicates, err := store.FindDuplicateHashes()
// Returns: map[contentHash][]storedHash, only groups with more than one hash.
// Transaction hashes are metadata and never appear in this map.
```

## Data Integrity Guarantees

### Fingerprint Verification

Each snapshot includes a **fingerprint** - a SHA-256 hash of its ledger entries:

```go
snap.Fingerprint = ComputeFingerprint(snap)
```

This fingerprint is:
- Computed deterministically by hashing sorted (key, value) pairs
- Verified by `Load()` (logs `DRIFT DETECTED: ...`) and by `ValidateSnapshotBeforeReplay()` (hard error)
- Returned as `*DriftWarning` by `LoadWithDiagnostics()`; `LoadPersisted()` and `RecoverFromCorruption()` do not verify it

### Atomic Writes

All writes use atomic rename to prevent partial/corrupted files:

```
1. Write to temporary file (.snap.json.tmp)
2. Verify write succeeded
3. Atomic rename to final path
4. Clean up temp file on error
```

If the process crashes between steps 1-2, the temp file is orphaned. The recovery process will ignore temp files and reconstruct the index from valid snapshots.

## Usage Patterns

### Pattern 1: Save and Deduplicate

```go
store := snapshot.NewDedupStore(cacheDir)

hash, isNew, err := store.SaveWithDedup(&ReplayMetadata{
    TxHash:        "abc123...",
    Network:       "testnet",
    EnvelopeXdr:   "...",
    ResultMetaXdr: "...",
}, snapshot)

if err != nil {
    log.Fatal(err)
}

if isNew {
    log.Printf("Stored new snapshot: %s", hash)
} else {
    log.Printf("Reused existing snapshot: %s", hash)
}
```

### Pattern 2: Batch Processing

```go
store := snapshot.NewDedupStore(cacheDir)
savedHashes := make(map[string]int)

for _, tx := range transactions {
    snap := replayTransaction(tx)
    hash, isNew, _ := store.SaveWithDedup(metadata, snap)
    if isNew {
        savedHashes[hash]++
    }
}

// After batch processing
count, _ := store.QueryDedup()
log.Printf("Stored %d unique snapshots out of %d transactions", 
           count, len(transactions))
```

### Pattern 3: Recovery After Crash

```go
store := snapshot.NewDedupStore(cacheDir)

// After finding corrupted files
recovered, err := store.RecoverFromCorruption()
if err != nil {
    log.Fatal(err)
}

log.Printf("Recovered %d valid snapshots", recovered)

// Resume normal operations
hash, isNew, _ := store.SaveWithDedup(meta, snap)
```

## Performance Characteristics

### Storage Overhead

**Before deduplication:**
- Each transaction → 1 file on disk
- 1000 transactions → 1000 files (duplicate snapshots take full space)

**After deduplication:**
- 1000 transactions with 800 unique snapshots → 800 files
- 20% reduction in storage usage

Average snapshot file size: 10-100 KB depending on ledger complexity.

### Lookup Speed

- **LoadByHash**: O(1) disk seek + JSON parse (milliseconds)
- **GetStoredHashes**: O(n) where n = number of unique snapshots (typically small)
- **FindDuplicateHashes**: O(n) with dedup store load

### Deduplication Overhead

- **ContentHash computation**: ~1-2ms per snapshot
- **Index update**: ~0.1ms per snapshot
- **Dedup check**: Filesystem stat (0.1ms typical)

For most use cases, the dedup overhead is negligible compared to network/simulation time.

## Storage Format

### PersistedSnapshot JSON Structure

```json
{
  "metadata": {
    "schema_version": 2,
    "glassbox_version": "v1.2.0",
    "saved_at": "2026-06-02T10:30:00Z",
    "tx_hash": "abc123...",
    "network": "testnet",
    "envelope_xdr": "base64...",
    "result_meta_xdr": "base64...",
    "source_hash": "sha256...",
    "param_fingerprint": "sha256...",
    "ledger_format": "base64-xdr"
  },
  "snapshot": {
    "ledgerEntries": [
      ["key1-base64", "val1-base64"],
      ["key2-base64", "val2-base64"]
    ],
    "linearMemory": "base64...",
    "fingerprint": "sha256hex..."
  }
}
```

### Index File Structure

```json
{
  "hashes": [
    "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789",
    "fedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210"
  ],
  "created_at": "",
  "updated_at": ""
}
```

`created_at` and `updated_at` are currently always empty strings: `updateIndex`
and `RecoverFromCorruption()` write only `hashes`.

## Error Handling

### Common Errors and Solutions

| Error | Cause | Solution |
|-------|-------|----------|
| `snapshot schema version %d is too old to load` / `snapshot schema version %d was produced by a newer version of Glassbox` | Glassbox upgraded, schema changed | Re-run debug command to regenerate snapshot; upgrade Glassbox for a future-version file |
| `snapshot fingerprint mismatch` | Ledger entries changed after save | Use `LoadWithDiagnostics()` / `ValidateSnapshotBeforeReplay()`; `RecoverFromCorruption()` does not check fingerprints |
| `failed to read snapshot file` | File deleted or permissions issue | Restore from backup or delete corrupted cache |
| `hash must not be empty` | Programming error (empty hash string) | Check ContentHash() return value |

## Testing

The deduplication system includes comprehensive test coverage:

- **Hash determinism**: Same snapshot → same hash
- **Deduplication verification**: Identical snapshots → single file
- **Recovery**: Corrupted files → skip and rebuild index
- **Isolation**: Multiple snapshots in single store → no interference
- **Edge cases**: Nil snapshots, empty hashes, timestamp variations
- **Boundary**: `TestContentHash_DedupBoundary_*` (identical collapse, minimal differences, ordering invariance, linear memory, count/ordering)

Run tests:

```bash
go test -v ./internal/snapshot/... -run Dedup
```

### Worked example: dedup hit and fingerprint-drift failure

Both outcomes are covered by checked-in tests, so this procedure can be
repeated on a clean checkout:

- Success: `TestDedupStore_SaveWithDedup_Deduplication` saves the same ledger
  state under different metadata; both saves return the same content hash, the
  second returns `isNew == false`, and `QueryDedup()` reports one unique file.
- Failure: `TestValidate_FingerprintMismatch` tampers with ledger state without
  recomputing the fingerprint. `LoadByHash` and `RecoverFromCorruption` use
  `LoadPersisted`, which does not check fingerprints; only
  `ValidateSnapshotBeforeReplay` reports `snapshot fingerprint mismatch`.


## Migration from Flat Storage

If migrating from the old flat storage (one file per transaction):

```go
// Option 1: Manual migration during next snapshot save
oldPath := "old_snapshot.json"
ps, _ := LoadPersisted(oldPath)
hash, _, _ := store.SaveWithDedup(ps.Metadata, ps.Snapshot)
// Old file can be deleted - content is preserved with dedup hash

// Option 2: Batch migration
files, _ := filepath.Glob("snapshots/*.json")
for _, file := range files {
    ps, _ := LoadPersisted(file)
    store.SaveWithDedup(ps.Metadata, ps.Snapshot)
}
```

`DedupStore` reads only content-addressed names under `snapshots/`:
`GetStoredHashes()` skips subdirectories and `LoadByHash()` expects
`<hash8>_<hash56>.snap.json`. `glassbox snapshot save` still writes
`snapshots/<network>/<txhash[:16]>.snap.json` via `DefaultSnapshotPath`, so
migrate those files explicitly as above.

## Best Practices

1. **Always use `SaveWithDedup()`** instead of direct `SavePersisted()` calls when building new code
2. **Check `isNew` flag** to monitor deduplication effectiveness
3. **Run `RecoverFromCorruption()` periodically** for long-running servers
4. **Use `GetDiskUsage()` in monitoring** to track storage growth
5. **Index file is optional** - reconstruct it anytime with `RecoverFromCorruption()`
6. **Never modify `.snap.json` files manually** - fingerprint validation will catch tampering

## Future Enhancements

Potential improvements to the deduplication system:

- [ ] Compression: Store deduplicated snapshots in compressed format
- [ ] LRU eviction: replace the current `glassbox compact` mark-and-sweep policy (see `internal/cmd/compact.go`)
- [ ] Diff storage: Store only differences between similar snapshots
- [ ] Distributed index: Share dedup index across multiple Glassbox instances
- [ ] Verification: Periodic integrity checks on stored snapshots

---

## Architecture Decision Records

The following ADRs govern the design decisions behind snapshot storage:

- [ADR-004: Data Classification and Cross-Boundary Data Flows](adr/004-data-classification.md) — classifies snapshot / ledger state as `INTERNAL` data, documents that it is stored locally only and not transmitted to external services, and records the atomic write and fingerprint controls at the local persistence boundary (Boundary E).
- [ADR-007: Offline Guarantees](adr/007-offline-guarantees.md) — documents the offline replay guarantee, the snapshot fingerprint verification on load, and why provenance metadata is excluded from the content hash.
- [ADR-003: Trust Boundaries and Component Trust Levels](adr/003-trust-boundaries.md) — places the snapshot store in the Tier-0 CLI host process and confirms that no signing credentials are stored in snapshot files.
