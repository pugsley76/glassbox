// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

package trace

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/dotandev/glassbox/internal/snapshot"
)

// StateChange represents a detected change in contract state
type StateChange struct {
	// ChangeType indicates the kind of change: "ledger_entry", "code_artifact", "memory"
	ChangeType string `json:"change_type"`
	// Key is the ledger entry key or artifact identifier (Base64 XDR)
	Key string `json:"key"`
	// OldValue is the previous value (Base64 XDR for ledger entries, hex for memory)
	OldValue string `json:"old_value,omitempty"`
	// NewValue is the updated value
	NewValue string `json:"new_value"`
	// AffectedSteps lists execution steps that depend on this state
	AffectedSteps []int `json:"affected_steps"`
	// ContractID identifies which contract is affected (if applicable)
	ContractID string `json:"contract_id,omitempty"`
}

// StateChangeDetector monitors contract state and identifies changes that require trace refresh
type StateChangeDetector struct {
	// baseSnapshot is the original ledger state
	baseSnapshot *snapshot.Snapshot
	// currentSnapshot is the latest state
	currentSnapshot *snapshot.Snapshot
	// stateFingerprints maps step numbers to state fingerprints for quick comparison
	stateFingerprints map[int]string
	// stepDependencies maps ledger keys to the steps that read them
	stepDependencies map[string][]int
}

// NewStateChangeDetector creates a detector initialized with the base snapshot
func NewStateChangeDetector(base *snapshot.Snapshot) *StateChangeDetector {
	return &StateChangeDetector{
		baseSnapshot:      base,
		currentSnapshot:   base,
		stateFingerprints: make(map[int]string),
		stepDependencies:  make(map[string][]int),
	}
}

// RecordStateDependency records that a specific step depends on a ledger entry
func (d *StateChangeDetector) RecordStateDependency(step int, ledgerKey string) {
	if d.stepDependencies[ledgerKey] == nil {
		d.stepDependencies[ledgerKey] = make([]int, 0)
	}
	d.stepDependencies[ledgerKey] = append(d.stepDependencies[ledgerKey], step)
}

// RecordStateFingerprint captures the state fingerprint at a specific step
func (d *StateChangeDetector) RecordStateFingerprint(step int, state *ExecutionState) {
	fp := d.computeStateFingerprint(state)
	d.stateFingerprints[step] = fp
}

// computeStateFingerprint creates a deterministic hash of an execution state
func (d *StateChangeDetector) computeStateFingerprint(state *ExecutionState) string {
	// Create a normalized representation for hashing
	normalized := map[string]interface{}{
		"operation":   state.Operation,
		"contract_id": state.ContractID,
		"function":    state.Function,
	}
	
	// Include host state keys sorted for determinism
	if len(state.HostState) > 0 {
		hostStateJSON, _ := json.Marshal(state.HostState)
		normalized["host_state"] = string(hostStateJSON)
	}
	
	data, _ := json.Marshal(normalized)
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// UpdateSnapshot updates the current snapshot and returns detected changes
func (d *StateChangeDetector) UpdateSnapshot(newSnapshot *snapshot.Snapshot) ([]StateChange, error) {
	if newSnapshot == nil {
		return nil, fmt.Errorf("new snapshot cannot be nil")
	}

	changes := make([]StateChange, 0)
	
	oldEntries := d.currentSnapshot.ToMap()
	newEntries := newSnapshot.ToMap()
	
	// Detect changed and new entries
	for key, newValue := range newEntries {
		oldValue, existed := oldEntries[key]
		if !existed || oldValue != newValue {
			change := StateChange{
				ChangeType:    "ledger_entry",
				Key:           key,
				OldValue:      oldValue,
				NewValue:      newValue,
				AffectedSteps: d.stepDependencies[key],
			}
			changes = append(changes, change)
		}
	}
	
	// Detect deleted entries
	for key, oldValue := range oldEntries {
		if _, exists := newEntries[key]; !exists {
			change := StateChange{
				ChangeType:    "ledger_entry",
				Key:           key,
				OldValue:      oldValue,
				NewValue:      "",
				AffectedSteps: d.stepDependencies[key],
			}
			changes = append(changes, change)
		}
	}
	
	// Check for linear memory changes
	oldMemory, _ := d.currentSnapshot.DecodeLinearMemory()
	newMemory, _ := newSnapshot.DecodeLinearMemory()
	if !bytesEqual(oldMemory, newMemory) {
		change := StateChange{
			ChangeType: "memory",
			Key:        "linear_memory",
			NewValue:   newSnapshot.LinearMemory,
			// Memory changes affect all steps (conservative approach)
			AffectedSteps: allSteps(d.stateFingerprints),
		}
		changes = append(changes, change)
	}
	
	d.currentSnapshot = newSnapshot
	return changes, nil
}

// DetectCodeArtifactChanges compares contract WASM bytecode between snapshots
func (d *StateChangeDetector) DetectCodeArtifactChanges(oldWasm, newWasm []byte, contractID string) (*StateChange, error) {
	if bytesEqual(oldWasm, newWasm) {
		return nil, nil // No change
	}
	
	oldHash := sha256.Sum256(oldWasm)
	newHash := sha256.Sum256(newWasm)
	
	change := &StateChange{
		ChangeType:    "code_artifact",
		Key:           fmt.Sprintf("contract:%s", contractID),
		OldValue:      hex.EncodeToString(oldHash[:]),
		NewValue:      hex.EncodeToString(newHash[:]),
		ContractID:    contractID,
		AffectedSteps: d.findStepsForContract(contractID),
	}
	
	return change, nil
}

// findStepsForContract returns all steps that involve a specific contract
func (d *StateChangeDetector) findStepsForContract(contractID string) []int {
	steps := make([]int, 0)
	// This will be populated by the trace analyzer
	// For now, return empty slice (will be enhanced when integrated with trace)
	return steps
}

// GetAffectedSteps returns all unique steps affected by the given changes
func GetAffectedSteps(changes []StateChange) []int {
	stepSet := make(map[int]bool)
	for _, change := range changes {
		for _, step := range change.AffectedSteps {
			stepSet[step] = true
		}
	}
	
	steps := make([]int, 0, len(stepSet))
	for step := range stepSet {
		steps = append(steps, step)
	}
	return steps
}

// ComputeRefreshRange determines the minimum range of steps that need re-simulation
// Returns (startStep, endStep) where both are inclusive
func ComputeRefreshRange(affectedSteps []int, totalSteps int) (int, int) {
	if len(affectedSteps) == 0 {
		return -1, -1
	}
	
	minStep := affectedSteps[0]
	maxStep := affectedSteps[0]
	
	for _, step := range affectedSteps {
		if step < minStep {
			minStep = step
		}
		if step > maxStep {
			maxStep = step
		}
	}
	
	// Expand range to include dependent steps (conservative approach)
	// If a change affects step N, we re-simulate from N to end
	return minStep, totalSteps - 1
}

// bytesEqual compares two byte slices for equality
func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// allSteps returns all step numbers from the fingerprint map
func allSteps(fingerprints map[int]string) []int {
	steps := make([]int, 0, len(fingerprints))
	for step := range fingerprints {
		steps = append(steps, step)
	}
	return steps
}
