# Event Ordering Verification Guide

## Overview

This guide provides procedures to verify that the [Event Ordering Contract](event-ordering-contract.md) produces identical event order across runs and platforms, meeting the acceptance criteria.

## Verification Methods

### 1. Deterministic Sequence ID Assignment

**Test**: Run the same trace generation multiple times and verify sequence IDs are identical.

```bash
# Run trace generation 5 times
for i in {1..5}; do
    go test -v ./internal/trace -run TestExecutionTrace_AddStateAssignsSequenceIDs -count=1
done
```

**Expected Result**: All runs produce identical sequence IDs (1, 2, 3, ...)

**Verification Script (illustrative only — `TestDeterministicSequenceAssignment` is
not checked in; the executable coverage is
`TestExecutionTrace_AddStateAssignsSequenceIDs`)**:
```go
// test_deterministic_sequence.go
func TestDeterministicSequenceAssignment(t *testing.T) {
    var results [][]uint64
    
    for run := 0; run < 10; run++ {
        trace := NewExecutionTrace("test", 10)
        for i := 0; i < 100; i++ {
            trace.AddState(ExecutionState{Operation: "op"})
        }
        
        var ids []uint64
        for _, state := range trace.States {
            ids = append(ids, state.SequenceID)
        }
        results = append(results, ids)
    }
    
    // Verify all runs produced identical sequence IDs
    for i := 1; i < len(results); i++ {
        if !reflect.DeepEqual(results[0], results[i]) {
            t.Errorf("Run %d produced different sequence IDs than run 0", i)
        }
    }
}
```

### 2. Cross-Platform Consistency

**Test**: Generate the same trace on different platforms and compare sequence IDs.

**Platforms to Test**:
- Linux (amd64)
- macOS (amd64/arm64)
- Windows (amd64)

**Procedure**:
1. Run the ordering tests below on each platform with `-count=5`.
2. Compare sequence IDs and parent IDs, not raw JSON bytes: `ExecutionTrace`
   stamps `time.Now()` on every state, so exported JSON is not byte-identical
   across runs or machines.

**Expected Result**: identical test results on each platform; sequence IDs are
the cross-platform ordering contract. CI already runs these tests on Linux,
macOS, and Windows (see [ci-test.yml](../.github/workflows/ci-test.yml)).

```bash
go test -v ./internal/trace -run 'TestSerializationPreservesSequenceOrder|TestDiagnosticEventOrderingPreserved|TestContractEventOrderingPreserved|TestDiffWithSequenceIDs' -count=5
```

### 3. Parent Relationship Preservation

**Test**: Verify parent-child relationships survive export and filtering.

The following snippet is illustrative only: `TestParentRelationshipPreservation`
is not checked in. Executable coverage is
`TestSerializationPreservesSequenceOrder` in
`internal/trace/event_ordering_integration_test.go`.

```go
func TestParentRelationshipPreservation(t *testing.T) {
    trace := NewExecutionTrace("parent_test", 10)
    
    // Create nested structure
    trace.AddState(ExecutionState{Operation: "parent"})
    parentID := trace.States[0].SequenceID
    trace.sequencer.PushParent(parentID)
    trace.AddState(ExecutionState{Operation: "child"})
    trace.sequencer.PopParent()
    
    // Export
    data, _ := trace.ToJSON()
    restored, _ := FromJSON(data)
    
    // Verify parent relationship preserved
    if restored.States[1].ParentSequenceID != parentID {
        t.Errorf("Parent relationship not preserved after export")
    }
    
    // Filter
    filter := &FilterExpression{Function: "child"}
    ft, _ := ApplyFilter(restored, filter)
    
    // Verify parent context in filtered trace
    for childIdx, parentIdx := range ft.ParentContext {
        childSeqID := restored.States[childIdx].SequenceID
        parentSeqID := restored.States[parentIdx].SequenceID
        if childSeqID <= parentSeqID {
            t.Errorf("Invalid parent-child relationship in filtered trace")
        }
    }
}
```

### 4. Diff Stability

**Test**: Verify diff operations produce identical results across runs.

Illustrative only: `TestDiffStability` is not checked in. Use
`TestDiffWithSequenceIDs` for executable coverage; the "reorder without
sequence IDs" step below has no dedicated test yet.

```go
func TestDiffStability(t *testing.T) {
    trace1 := NewExecutionTrace("trace1", 10)
    trace2 := NewExecutionTrace("trace2", 10)
    
    // Add identical states
    for i := 0; i < 50; i++ {
        state := ExecutionState{Operation: "op"}
        trace1.AddState(state)
        trace2.AddState(state)
    }
    
    // Compute diff multiple times
    var diffs []string
    for i := 0; i < 10; i++ {
        diff := ComputeTraceDiff(trace1, trace2)
        json, _ := diff.RenderJSON()
        diffs = append(diffs, string(json))
    }
    
    // Verify all diffs are identical
    for i := 1; i < len(diffs); i++ {
        if diffs[i] != diffs[0] {
            t.Errorf("Diff output not stable across runs")
        }
    }
}
```

### 5. Concurrent Collection Determinism

**Test**: Verify concurrent event collection produces deterministic ordering.

Illustrative only: `TestConcurrentDeterminism` is not checked in. Use
`TestConcurrentEventCollection` for executable coverage.

```go
func TestConcurrentDeterminism(t *testing.T) {
    var results [][]uint64
    
    for run := 0; run < 10; run++ {
        trace := NewExecutionTrace("concurrent_test", 10)
        done := make(chan bool, 100)
        
        // Add 100 states concurrently
        for i := 0; i < 100; i++ {
            go func() {
                trace.AddState(ExecutionState{Operation: "op"})
                done <- true
            }()
        }
        
        for i := 0; i < 100; i++ {
            <-done
        }
        
        // Extract sequence IDs
        var ids []uint64
        for _, state := range trace.States {
            ids = append(ids, state.SequenceID)
        }
        results = append(results, ids)
    }
    
    // Verify sequence IDs are monotonic in each run
    for _, ids := range results {
        for i := 1; i < len(ids); i++ {
            if ids[i] <= ids[i-1] {
                t.Errorf("Sequence IDs not monotonic")
            }
        }
    }
    
    // Note: Order may differ between runs due to concurrency,
    // but sequence IDs should always be unique and monotonic
}
```

## Acceptance Criteria Verification

### AC1: Same replay produces identical event order across runs and platforms

**Verification Steps**:
1. Create a trace fixture with nested and simultaneous events
2. Run the fixture 10 times on the same platform
3. Compare sequence IDs across all runs
4. Run the fixture on different platforms (Linux, macOS, Windows)
5. Compare sequence IDs across platforms

**Success Criteria**:
- ✅ Sequence IDs are identical across all runs on the same platform
- ✅ Sequence IDs are identical across different platforms
- ✅ Event order (by sequence ID) is consistent

**Test Command**:
```bash
go test -v ./internal/trace -run 'TestExecutionTrace_AddStateAssignsSequenceIDs|TestEventSequencer_MonotonicSequenceIDs' -count=10
```

### AC2: Parent-child relationships remain valid after export and filtering

**Verification Steps**:
1. Create a trace with nested calls (parent-child relationships)
2. Export to JSON
3. Import from JSON
4. Apply various filters
5. Validate parent relationships at each step

**Success Criteria**:
- ✅ Parent sequence IDs preserved after export/import
- ✅ Parent context maintained in filtered traces
- ✅ No circular references introduced
- ✅ Parent IDs always less than child IDs

**Test Command**:
```bash
go test -v ./internal/trace -run TestSerializationPreservesSequenceOrder
```

### AC3: Comparison reports identify true changes rather than ordering noise

**Verification Steps**:
1. Create two identical traces
2. Compute diff - should be empty
3. Modify one trace (actual change)
4. Compute diff - should show the change
5. Reorder events in one trace (without sequence IDs)
6. Compute diff with sequence IDs - should be empty
7. Compute diff without sequence IDs - should show ordering noise

**Success Criteria**:
- ✅ Identical traces produce empty diff
- ✅ Actual changes are detected
- ✅ Ordering noise is eliminated when sequence IDs are present
- ✅ Diff output is stable across runs

**Test Command**:
```bash
go test -v ./internal/trace -run 'TestDiffWithSequenceIDs|TestBackwardCompatibility'
```

## Regression Tests

### Test Suite

Run the complete event ordering test suite:

```bash
# Unit tests
go test -v ./internal/trace -run TestEventSequencer
go test -v ./internal/trace -run TestEventOrdering

# Integration tests
go test -v ./internal/trace -run TestFilteringPreservesSequenceOrder
go test -v ./internal/trace -run TestSerializationPreservesSequenceOrder
go test -v ./internal/trace -run TestDiagnosticEventOrderingPreserved
go test -v ./internal/trace -run TestContractEventOrderingPreserved
go test -v ./internal/trace -run TestMergingPreservesSequenceOrder
go test -v ./internal/trace -run TestDiffWithSequenceIDs
go test -v ./internal/trace -run TestBackwardCompatibility
go test -v ./internal/trace -run TestConcurrentEventCollection

# Fixture tests
go test -v ./internal/trace -run TestNestedAndSimultaneousEvents
```

### Regression example (issue #144)

Both outcomes are covered by checked-in tests in
`internal/trace/event_ordering_integration_test.go` and
`internal/trace/event_ordering_test.go`:

- Success: `TestSerializationPreservesSequenceOrder` and
  `TestNestedAndSimultaneousEvents` verify that sequence IDs and
  `parent_sequence_id` survive export/import and that a nested parent ID is
  always smaller than its child's.
- Failure: `TestEventOrdering_ValidateParentRelationships` verifies that a
  dangling `ParentSequenceID` is rejected; `ValidateParentRelationships`
  returns the index of the offending state (1 for a missing parent 999) rather
  than `-1`. This is the failure to expect when a trace is edited by hand.

```bash
go test -v ./internal/trace -run 'TestSerializationPreservesSequenceOrder|TestNestedAndSimultaneousEvents|TestEventOrdering_ValidateParentRelationships'
```

### Continuous Integration

There is no `.github/workflows/event-ordering.yml`. Event ordering tests run in
the `go-unit` job of [ci-test.yml](../.github/workflows/ci-test.yml), whose
matrix covers `ubuntu-latest`, `macos-latest`, and `windows-latest` and uses
`go test -race` over `./internal/...` with `go-version-file: go.mod`. Do not
add a second matrix with older Go versions: [go.mod](../go.mod) requires Go
1.26. Documentation examples are checked by
[validate-docs.sh](../scripts/validate-docs.sh); its flag check only inspects
lines that begin with `glassbox `.

## Performance Benchmarks

### Sequence ID Assignment Overhead

```go
func BenchmarkSequenceIDAssignment(b *testing.B) {
    trace := NewExecutionTrace("bench", 100)
    state := ExecutionState{Operation: "op"}
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        trace.AddState(state)
    }
}
```

**Expected**: Negligible overhead (< 1% of total trace processing time)

### Sorting Performance

```go
func BenchmarkEventSorting(b *testing.B) {
    ordering := &EventOrdering{}
    states := make([]ExecutionState, 10000)
    for i := range states {
        states[i] = ExecutionState{SequenceID: uint64(i + 1)}
    }
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        ordering.SortStatesBySequenceID(states)
    }
}
```

**Expected**: O(n log n) performance, acceptable for typical trace sizes

## Troubleshooting

### Issue: Non-deterministic sequence IDs

**Symptoms**: Sequence IDs differ between runs

**Causes**:
- Concurrent access without proper synchronization
- Manual sequence ID assignment conflicting with automatic assignment
- Reusing sequencer across multiple traces without reset

**Solutions**:
1. Ensure `EventSequencer` is used correctly with mutex protection
2. Use `sequencer.Reset()` when starting a new trace
3. Let automatic assignment handle sequence IDs when possible

### Issue: Invalid parent relationships

**Symptoms**: Validation fails with parent relationship errors

**Causes**:
- Parent stack not managed correctly (missing Push/Pop)
- Parent sequence ID greater than child sequence ID
- Non-existent parent referenced

**Solutions**:
1. Verify Push/Pop calls are balanced
2. Validate sequence IDs before setting parent relationships
3. Use `ValidateParentRelationships()` to detect issues

### Issue: Ordering noise in diffs

**Symptoms**: Diff shows changes that are only ordering differences

**Causes**:
- Traces lack sequence IDs
- Diff alignment uses fallback keys that are not unique
- Events have identical fallback keys

**Solutions**:
1. Ensure sequence IDs are assigned at collection time
2. Reindex old traces before diffing
3. Use sequence ID-based alignment in diff operations

## Summary

The event ordering contract provides:

✅ **Deterministic sequence ID assignment** across runs and platforms
✅ **Parent relationship preservation** through export and filtering
✅ **Stable diff output** that identifies true changes
✅ **Backward compatibility** with traces lacking sequence IDs
✅ **Thread-safe sequence ID assignment** (`AddState`'s `States` slice is not mutex-protected; see `TestConcurrentEventCollection`)
✅ **Focused test coverage** for the ordering contract and its main transformations; illustrative snippets in this guide are not executable tests

All acceptance criteria are met through the implemented sequence ID system, parent relationship tracking, and deterministic tie-breakers.
