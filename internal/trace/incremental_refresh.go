// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

package trace

import (
	"context"
	"fmt"
	"time"

	"github.com/dotandev/glassbox/internal/simulator"
	"github.com/dotandev/glassbox/internal/snapshot"
)

// RefreshRequest describes an incremental trace refresh operation
type RefreshRequest struct {
	// OriginalTrace is the existing trace to be refreshed
	OriginalTrace *ExecutionTrace
	// UpdatedSnapshot contains the new ledger state
	UpdatedSnapshot *snapshot.Snapshot
	// Changes lists the detected state changes
	Changes []StateChange
	// StartStep is the first step to re-simulate (inclusive)
	StartStep int
	// EndStep is the last step to re-simulate (inclusive)
	EndStep int
	// PreserveUnaffected keeps unaffected nodes unchanged
	PreserveUnaffected bool
}

// RefreshResult contains the outcome of an incremental refresh
type RefreshResult struct {
	// UpdatedTrace is the trace with refreshed steps
	UpdatedTrace *ExecutionTrace
	// RefreshedSteps lists which steps were actually re-simulated
	RefreshedSteps []int
	// PreservedSteps lists which steps were kept unchanged
	PreservedSteps []int
	// Duration is how long the refresh took
	Duration time.Duration
	// Success indicates whether the refresh completed without errors
	Success bool
	// Error contains any error encountered during refresh
	Error error
}

// IncrementalRefresher handles partial trace re-simulation
type IncrementalRefresher struct {
	runner   simulator.RunnerInterface
	detector *StateChangeDetector
}

// NewIncrementalRefresher creates a refresher with the given simulator runner
func NewIncrementalRefresher(runner simulator.RunnerInterface) *IncrementalRefresher {
	return &IncrementalRefresher{
		runner: runner,
	}
}

// SetDetector assigns a state change detector to the refresher
func (r *IncrementalRefresher) SetDetector(detector *StateChangeDetector) {
	r.detector = detector
}

// Refresh performs an incremental trace refresh based on the request
func (r *IncrementalRefresher) Refresh(ctx context.Context, req *RefreshRequest) (*RefreshResult, error) {
	startTime := time.Now()
	
	if req.OriginalTrace == nil {
		return nil, fmt.Errorf("original trace cannot be nil")
	}
	if req.UpdatedSnapshot == nil {
		return nil, fmt.Errorf("updated snapshot cannot be nil")
	}
	if req.StartStep < 0 || req.StartStep >= len(req.OriginalTrace.States) {
		return nil, fmt.Errorf("invalid start step: %d", req.StartStep)
	}
	if req.EndStep < req.StartStep || req.EndStep >= len(req.OriginalTrace.States) {
		return nil, fmt.Errorf("invalid end step: %d", req.EndStep)
	}
	
	result := &RefreshResult{
		RefreshedSteps: make([]int, 0),
		PreservedSteps: make([]int, 0),
		Success:        false,
	}
	
	// Create a new trace with preserved metadata
	refreshedTrace := &ExecutionTrace{
		TransactionHash:  req.OriginalTrace.TransactionHash,
		StartTime:        req.OriginalTrace.StartTime,
		States:           make([]ExecutionState, len(req.OriginalTrace.States)),
		Snapshots:        make([]StateSnapshot, 0),
		DiagnosticEvents: req.OriginalTrace.DiagnosticEvents,
		Annotations:      req.OriginalTrace.Annotations,
		CurrentStep:      req.OriginalTrace.CurrentStep,
		SnapshotInterval: req.OriginalTrace.SnapshotInterval,
	}
	
	// Copy states before the refresh range (preserved)
	for i := 0; i < req.StartStep; i++ {
		refreshedTrace.States[i] = req.OriginalTrace.States[i]
		result.PreservedSteps = append(result.PreservedSteps, i)
	}
	
	// Re-simulate the affected range
	for step := req.StartStep; step <= req.EndStep; step++ {
		originalState := &req.OriginalTrace.States[step]
		
		// Check if this step is actually affected by changes
		if req.PreserveUnaffected && !r.isStepAffected(step, req.Changes) {
			// Preserve unchanged step
			refreshedTrace.States[step] = *originalState
			result.PreservedSteps = append(result.PreservedSteps, step)
			continue
		}
		
		// Re-simulate this step with updated snapshot
		newState, err := r.reSimulateStep(ctx, originalState, req.UpdatedSnapshot)
		if err != nil {
			result.Error = fmt.Errorf("failed to re-simulate step %d: %w", step, err)
			result.Duration = time.Since(startTime)
			return result, result.Error
		}
		
		refreshedTrace.States[step] = *newState
		result.RefreshedSteps = append(result.RefreshedSteps, step)
	}
	
	// Copy states after the refresh range (preserved)
	for i := req.EndStep + 1; i < len(req.OriginalTrace.States); i++ {
		refreshedTrace.States[i] = req.OriginalTrace.States[i]
		result.PreservedSteps = append(result.PreservedSteps, i)
	}
	
	// Rebuild snapshots for the refreshed trace
	r.rebuildSnapshots(refreshedTrace)
	
	refreshedTrace.EndTime = time.Now()
	result.UpdatedTrace = refreshedTrace
	result.Duration = time.Since(startTime)
	result.Success = true
	
	return result, nil
}

// isStepAffected checks if a step is affected by any of the changes
func (r *IncrementalRefresher) isStepAffected(step int, changes []StateChange) bool {
	for _, change := range changes {
		for _, affectedStep := range change.AffectedSteps {
			if affectedStep == step {
				return true
			}
		}
	}
	return false
}

// reSimulateStep re-executes a single step with the updated snapshot
func (r *IncrementalRefresher) reSimulateStep(ctx context.Context, originalState *ExecutionState, updatedSnapshot *snapshot.Snapshot) (*ExecutionState, error) {
	// For now, we clone the original state with updated timestamp
	// In a full implementation, this would invoke the simulator with the new snapshot
	newState := &ExecutionState{
		Step:             originalState.Step,
		Timestamp:        time.Now(),
		Operation:        originalState.Operation,
		EventType:        originalState.EventType,
		ContractID:       originalState.ContractID,
		Function:         originalState.Function,
		ContractMetadata: originalState.ContractMetadata,
		Arguments:        originalState.Arguments,
		RawArguments:     originalState.RawArguments,
		ReturnValue:      originalState.ReturnValue,
		RawReturnValue:   originalState.RawReturnValue,
		Error:            originalState.Error,
		HostState:        make(map[string]interface{}),
		Memory:           make(map[string]interface{}),
		WasmInstruction:  originalState.WasmInstruction,
		SourceFile:       originalState.SourceFile,
		SourceLine:       originalState.SourceLine,
		GitHubLink:       originalState.GitHubLink,
		Cost:             originalState.Cost,
	}
	
	// Copy host state and memory
	for k, v := range originalState.HostState {
		newState.HostState[k] = v
	}
	for k, v := range originalState.Memory {
		newState.Memory[k] = v
	}
	
	// TODO: In full implementation, construct SimulationRequest with updatedSnapshot
	// and call r.runner.Run(ctx, simReq) to get actual new state
	
	return newState, nil
}

// rebuildSnapshots reconstructs snapshots for the refreshed trace
func (r *IncrementalRefresher) rebuildSnapshots(trace *ExecutionTrace) {
	trace.Snapshots = make([]StateSnapshot, 0)
	
	for i, state := range trace.States {
		if i%trace.SnapshotInterval == 0 {
			snapshot := StateSnapshot{
				Step:      state.Step,
				Timestamp: state.Timestamp,
				HostState: make(map[string]interface{}),
				Memory:    make(map[string]interface{}),
				CallStack: []string{},
				built:     false,
			}
			trace.Snapshots = append(trace.Snapshots, snapshot)
		}
	}
}

// QuickRefresh performs a fast refresh by only updating states with detected changes
// This is more efficient than Refresh but requires accurate change detection
func (r *IncrementalRefresher) QuickRefresh(ctx context.Context, trace *ExecutionTrace, changes []StateChange) (*RefreshResult, error) {
	affectedSteps := GetAffectedSteps(changes)
	if len(affectedSteps) == 0 {
		return &RefreshResult{
			UpdatedTrace:   trace,
			RefreshedSteps: []int{},
			PreservedSteps: allStepNumbers(trace),
			Success:        true,
		}, nil
	}
	
	startStep, endStep := ComputeRefreshRange(affectedSteps, len(trace.States))
	if startStep < 0 {
		return &RefreshResult{
			UpdatedTrace:   trace,
			RefreshedSteps: []int{},
			PreservedSteps: allStepNumbers(trace),
			Success:        true,
		}, nil
	}
	
	req := &RefreshRequest{
		OriginalTrace:      trace,
		UpdatedSnapshot:    r.detector.currentSnapshot,
		Changes:            changes,
		StartStep:          startStep,
		EndStep:            endStep,
		PreserveUnaffected: true,
	}
	
	return r.Refresh(ctx, req)
}

// allStepNumbers returns a slice of all step numbers in the trace
func allStepNumbers(trace *ExecutionTrace) []int {
	steps := make([]int, len(trace.States))
	for i := range trace.States {
		steps[i] = i
	}
	return steps
}
