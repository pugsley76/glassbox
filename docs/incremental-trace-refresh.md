# Incremental Trace Refresh

## Overview

The **Incremental Trace Refresh** feature enables efficient trace updates during interactive debugging sessions when contract state changes. Instead of rerunning the entire replay, the refresher re-simulates a computed range and preserves the rest of the trace.

## Motivation

During interactive debugging, developers may:
- Modify ledger entries to test different scenarios
- Update contract code to fix bugs
- Change WASM linear memory state
- Inject different transaction parameters

Previously, any state change required a complete trace regeneration, which could take several seconds or minutes for complex transactions. Incremental refresh dramatically reduces this overhead by:
- **Detecting exactly which state changed** (ledger entries, code artifacts, memory)
- **Identifying affected trace steps** based on dependency analysis
- **Re-simulating only the minimum necessary range** of execution steps
- **Preserving unchanged portions** of the trace to minimize computation

## Architecture

### Components

#### 1. State Change Detector (`state_change_detector.go`)

Monitors contract state and identifies changes that require trace refresh.

**Key Features:**
- **Ledger Entry Tracking**: Detects modifications, additions, and deletions of ledger entries
- **Code Artifact Monitoring**: Identifies WASM bytecode changes via SHA-256 fingerprinting
- **Dependency Recording**: Maps which execution steps depend on which ledger keys
- **State Fingerprinting**: Creates deterministic hashes for quick state comparison

**Usage Example:**
```go
// Initialize detector with base snapshot
baseSnapshot := loadSnapshot("base.json")
detector := NewStateChangeDetector(baseSnapshot)

// Record dependencies during trace execution
detector.RecordStateDependency(stepNumber, ledgerKey)
detector.RecordStateFingerprint(stepNumber, state)

// Update with new snapshot and detect changes
newSnapshot := loadSnapshot("updated.json")
changes, err := detector.UpdateSnapshot(newSnapshot)
```

#### 2. Incremental Refresher (`incremental_refresh.go`)

Handles partial trace re-simulation based on detected changes.

**Key Features:**
- **Selective Re-simulation**: Only recomputes affected execution steps
- **Preservation Mode**: Optionally keeps unaffected steps unchanged
- **Snapshot Rebuilding**: Reconstructs lazy snapshots after refresh
- **Performance Metrics**: Tracks refresh duration and step counts

**Usage Example:**
```go
refresher := NewIncrementalRefresher(simulatorRunner)
refresher.SetDetector(detector)

req := &RefreshRequest{
    OriginalTrace:      existingTrace,
    UpdatedSnapshot:    newSnapshot,
    Changes:            detectedChanges,
    StartStep:          10,
    EndStep:            50,
    PreserveUnaffected: true,
}

result, err := refresher.Refresh(ctx, req)
if result.Success {
    fmt.Printf("Refreshed %d steps in %v\n", 
        len(result.RefreshedSteps), result.Duration)
}
```

#### 3. Viewer Refresh Handler (`viewer_refresh.go`)

Integrates incremental refresh into the interactive trace viewer.

**Key Features:**
- **Command Integration**: Adds `refresh` command to the interactive viewer
- **Auto-refresh Mode**: Optionally refreshes automatically on state changes
- **Viewport Optimization**: Can refresh only the visible portion of the trace
- **User Feedback**: Displays progress and summary of refresh operations

## Usage

### Interactive Viewer Commands

The viewer reads commands at its `>` prompt; there is no `:` command mode.
`refresh` and `refresh-status` (`rs`) are registered in `internal/trace/viewer.go`,
but they only work after a caller attaches a `ViewerRefreshHandler` with
`InteractiveViewer.AttachRefreshHandler` before `Start()`. The CLI
`glassbox trace <trace-file>` path does not attach one.

#### Basic Refresh

```text
> refresh /path/to/updated_snapshot.json
```

This will:
1. Load the updated snapshot
2. Detect all state changes
3. Identify affected execution steps
4. Re-simulate the computed range
5. Update the viewer with refreshed trace

A successful refresh returns to the viewer; it does not produce a process exit
code. The most important failure is missing dependencies (see
[Scenario](#scenario-successful-refresh-and-the-most-important-failure)).

#### View Refresh Status

```text
> refresh-status
> rs
```

Displays:
- Auto-refresh enabled/disabled
- Time of last refresh

Step counts are printed by `refresh` itself; `refresh-status` does not store
them.

### Programmatic Usage

```go
// Create viewer with refresh support
viewer := NewInteractiveViewer(trace)
refreshHandler := NewViewerRefreshHandler(viewer, refresher)
refreshHandler.InitializeWithSnapshot(baseSnapshot)

// Enable auto-refresh mode
refreshHandler.EnableAutoRefresh()

// Perform refresh
ctx := context.Background()
updatedSnapshot := loadSnapshot("new_state.json")
err := refreshHandler.RefreshTrace(ctx, updatedSnapshot)
```

### API Integration

```go
// In your debug command or tool
import "github.com/dotandev/glassbox/internal/trace"

// Setup
detector := trace.NewStateChangeDetector(baseSnapshot)
refresher := trace.NewIncrementalRefresher(simulatorRunner)
refresher.SetDetector(detector)

// Detect changes
changes, _ := detector.UpdateSnapshot(newSnapshot)

// Quick refresh (automatic range computation)
result, err := refresher.QuickRefresh(ctx, existingTrace, changes)
```

## Performance Characteristics

### Time Complexity

- **Full Trace Regeneration**: O(n) where n is the total number of execution steps
- **Incremental Refresh**: O(n) for range computation, state copying, and snapshot rebuilding; O(k) simulator invocations when `PreserveUnaffected` is true and `k` steps are recorded as affected
- **Change Detection**: O(m) where m is the number of ledger entries

`ComputeRefreshRange` currently returns `[min(affectedSteps), totalSteps-1]`.
A change at step 10 in a 1000-step trace therefore selects steps 10-999;
`PreserveUnaffected: true` avoids re-simulating steps inside that range that
have no recorded dependency, but the range scan itself is O(n). No benchmark is
checked in for this feature, so this document makes no measured speed claim.

### Space Complexity

- **State Fingerprints**: O(n) - one hash per execution step
- **Dependency Map**: O(d) where d is the number of distinct ledger keys accessed
- **Preserved States**: O(n-k) - unaffected states kept in memory

## Implementation Details

### Dependency Tracking

The system tracks which execution steps depend on which ledger entries:

```go
// During trace generation
for step, state := range trace.States {
    for key := range state.HostState {
        detector.RecordStateDependency(step, key)
    }
}
```

When a ledger entry changes, all dependent steps are marked for refresh.

### State Fingerprinting

Each execution state gets a deterministic fingerprint:

```go
fingerprint := SHA256({
    operation: state.Operation,
    contract_id: state.ContractID,
    function: state.Function,
    host_state: JSON(state.HostState),
})
```

This enables quick equality checks and drift detection.

### Refresh Range Computation

The minimum range of steps to re-simulate is computed conservatively:

```go
// If steps 10, 15, and 20 are affected:
// Refresh range = [10, totalSteps-1]
// (from first affected to end, to capture cascading effects)

startStep = min(affectedSteps)
endStep = totalSteps - 1
```

This ensures correctness even when changes cascade through dependent steps.

### Snapshot Rebuilding

After refresh, lazy snapshots are reconstructed:

```go
// Rebuild snapshots at interval boundaries (default: every 100 steps)
for i := 0; i < len(refreshedTrace.States); i += snapshotInterval {
    snapshot := createLazySnapshot(i)
    refreshedTrace.Snapshots = append(refreshedTrace.Snapshots, snapshot)
}
```

## Testing

### Unit Tests

Run the comprehensive test suite:

```bash
go test ./internal/trace -run 'TestIncrementalRefresh|TestRefresh|TestQuickRefresh|TestRebuildSnapshots' -v
go test ./internal/trace -run 'Test(NewStateChangeDetector|RecordState|UpdateSnapshot|DetectCodeArtifact|GetAffectedSteps|ComputeRefreshRange)' -v
```

### Scenario: successful refresh and the most important failure

The programmatic path exercised by `internal/trace/incremental_refresh_integration_test.go`
records the ledger dependency on the same detector used by the refresher:

```go
baseSnapshot := snapshot.FromMap(map[string]string{"ledger_key": "old"})
updatedSnapshot := snapshot.FromMap(map[string]string{"ledger_key": "new"})
detector := trace.NewStateChangeDetector(baseSnapshot)
detector.RecordStateDependency(10, "ledger_key")
refresher := trace.NewIncrementalRefresher(runner)
refresher.SetDetector(detector)

changes, err := detector.UpdateSnapshot(updatedSnapshot)
if err != nil { /* handle */ }
result, err := refresher.QuickRefresh(ctx, existingTrace, changes)
if err != nil { /* handle */ }
fmt.Printf("success=%v refreshed=%v preserved=%v\n",
    result.Success, result.RefreshedSteps, result.PreservedSteps)
```

For a 30-step trace with only step 10 recorded, the output shape is:

```text
success=true refreshed=[10] preserved=[0 1 ... 9 11 ... 29]
```

The most important failure path is the viewer flow without those dependencies.
`ViewerRefreshHandler.InitializeWithSnapshot` seeds fingerprints only; it does
not call `RecordStateDependency`. A ledger-key change then produces no affected
steps, `ComputeRefreshRange` returns `-1, -1`, and the viewer prints:

```text
[FAIL] Refresh failed: refresh failed: invalid start step: -1
```

With no handler attached at all, the earlier failure is:

```text
[FAIL] Incremental refresh is not available in this session.
  Hint: start Glassbox with a base snapshot to enable refresh.
```

## Limitations and Future Work

### Current Limitations

1. **Conservative Range**: Currently refreshes from first affected step to end. Future optimization could use more precise dependency graphs.

2. **Memory Changes**: Linear memory changes trigger full trace refresh. Future work could track memory access patterns.

3. **No Partial Re-simulation**: Currently re-simulates complete steps. Future work could enable sub-step granularity.

### Planned Enhancements

- **Fine-grained Dependency Analysis**: Track exact memory addresses and storage keys accessed by each step
- **Parallel Refresh**: Re-simulate independent ranges in parallel for better performance
- **Diff Visualization**: Show visual diff of trace before/after refresh
- **Undo/Redo**: Allow reverting refreshes and maintaining refresh history
- **Smart Caching**: Cache intermediate simulation states for faster refreshes
- **Background Refresh**: Perform refresh asynchronously while user continues debugging

## Examples

### Example 1: Ledger Entry Modification

```go
// Initial state: balance = 1000
baseSnapshot := snapshot.FromMap(map[string]string{
    "balance_key": encodeXDR(1000),
})

// User modifies balance to 2000
updatedSnapshot := snapshot.FromMap(map[string]string{
    "balance_key": encodeXDR(2000),
})

// Refresh trace
changes, _ := detector.UpdateSnapshot(updatedSnapshot)
result, _ := refresher.QuickRefresh(ctx, trace, changes)

// Output: Refreshed 25 steps in 150ms
```

### Example 2: Contract Code Update

```go
// Detect WASM changes
oldWasm := loadContractWASM("v1.wasm")
newWasm := loadContractWASM("v2.wasm")

// DetectCodeArtifactChanges returns a StateChange, but findStepsForContract
// currently returns an empty affected-step set, so QuickRefresh will not
// re-simulate anything from this change alone.
change, _ := detector.DetectCodeArtifactChanges(
    oldWasm, newWasm, "CONTRACT_ID",
)

// Not currently wired: reSimulateStep sends only EnvelopeXdr, LedgerEntries,
// and SkipSourceMapping to the simulator, not the new WASM bytes.
result, _ := refresher.QuickRefresh(ctx, trace, []StateChange{*change})
```

### Example 3: Viewport-Only Refresh

```go
// RefreshCurrentView is programmatic only; no interactive command calls it.
handler := NewViewerRefreshHandler(viewer, refresher)
err := handler.RefreshCurrentView(ctx, updatedSnapshot)

// It filters to ~20 visible steps, but it is not exposed in viewer help and
// has no dedicated test.
```

## Best Practices

1. **Initialize Detector Early**: Set up the detector when creating the initial trace to capture all dependencies.

2. **Use QuickRefresh**: Let the system compute the optimal refresh range automatically.

3. **Enable PreserveUnaffected**: Keep unchanged steps to minimize computation.

4. **Monitor Performance**: Check `RefreshResult.Duration` to identify slow refreshes.

5. **Batch Changes**: If making multiple state modifications, batch them into one refresh operation.

6. **Verify Results**: After refresh, spot-check key execution points to ensure correctness.

## Troubleshooting

### Refresh Takes Longer Than Expected

- Check the refresh range: `result.RefreshedSteps`
- Verify snapshot size isn't excessively large
- Consider using viewport refresh for large traces

### Changes Not Reflected in Trace

- Ensure dependencies were recorded during initial trace generation
- Verify the updated snapshot is correctly formatted
- Check that affected steps are within the trace bounds

### Memory Usage High

- Reduce snapshot interval to free up memory
- Use viewport refresh instead of full trace refresh
- Clear old traces before performing refresh

## See Also

- [Interactive Trace Viewer](./trace-profiling.md)
- [Snapshot Deduplication](./snapshot-deduplication.md)
- [Session Bookmarking](./session-bookmarking.md)
- [Sandboxed Replay](./sandboxed-replay.md)

## Contributing

To contribute to incremental refresh:

1. Add tests in `internal/trace/*_test.go`
2. Update this documentation
3. Run full test suite: `go test ./internal/trace/...`
4. Benchmark performance: `go test -bench=. -benchmem`
5. Submit PR with detailed description

## License

Copyright 2026 Glassbox Users  
SPDX-License-Identifier: Apache-2.0
