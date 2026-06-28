// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

package trace

import (
	"testing"

	"github.com/dotandev/glassbox/internal/snapshot"
)

func TestNewStateChangeDetector(t *testing.T) {
	baseSnap := snapshot.FromMap(map[string]string{
		"key1": "value1",
		"key2": "value2",
	})
	
	detector := NewStateChangeDetector(baseSnap)
	
	if detector == nil {
		t.Fatal("expected non-nil detector")
	}
	if detector.baseSnapshot != baseSnap {
		t.Error("base snapshot not set correctly")
	}
	if len(detector.stepDependencies) != 0 {
		t.Error("expected empty step dependencies initially")
	}
}

func TestRecordStateDependency(t *testing.T) {
	detector := NewStateChangeDetector(nil)
	
	detector.RecordStateDependency(5, "ledger_key_1")
	detector.RecordStateDependency(10, "ledger_key_1")
	detector.RecordStateDependency(7, "ledger_key_2")
	
	deps1 := detector.stepDependencies["ledger_key_1"]
	if len(deps1) != 2 {
		t.Errorf("expected 2 dependencies for key1, got %d", len(deps1))
	}
	if deps1[0] != 5 || deps1[1] != 10 {
		t.Errorf("unexpected dependency steps: %v", deps1)
	}
	
	deps2 := detector.stepDependencies["ledger_key_2"]
	if len(deps2) != 1 || deps2[0] != 7 {
		t.Errorf("unexpected dependency steps for key2: %v", deps2)
	}
}

func TestRecordStateFingerprint(t *testing.T) {
	detector := NewStateChangeDetector(nil)
	
	state := &ExecutionState{
		Step:       5,
		Operation:  "contract_call",
		ContractID: "CTEST123",
		Function:   "transfer",
		HostState: map[string]interface{}{
			"balance": 1000,
		},
	}
	
	detector.RecordStateFingerprint(5, state)
	
	fp, exists := detector.stateFingerprints[5]
	if !exists {
		t.Error("fingerprint not recorded")
	}
	if fp == "" {
		t.Error("fingerprint should not be empty")
	}
	
	// Same state should produce same fingerprint
	detector.RecordStateFingerprint(6, state)
	fp2 := detector.stateFingerprints[6]
	if fp != fp2 {
		t.Error("identical states should produce identical fingerprints")
	}
}

func TestUpdateSnapshot_NoChanges(t *testing.T) {
	baseSnap := snapshot.FromMap(map[string]string{
		"key1": "value1",
		"key2": "value2",
	})
	
	detector := NewStateChangeDetector(baseSnap)
	
	// Update with identical snapshot
	sameSnap := snapshot.FromMap(map[string]string{
		"key1": "value1",
		"key2": "value2",
	})
	
	changes, err := detector.UpdateSnapshot(sameSnap)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	
	if len(changes) != 0 {
		t.Errorf("expected no changes, got %d", len(changes))
	}
}

func TestUpdateSnapshot_ModifiedEntry(t *testing.T) {
	baseSnap := snapshot.FromMap(map[string]string{
		"key1": "value1",
		"key2": "value2",
	})
	
	detector := NewStateChangeDetector(baseSnap)
	detector.RecordStateDependency(5, "key1")
	detector.RecordStateDependency(10, "key1")
	
	// Update with modified entry
	newSnap := snapshot.FromMap(map[string]string{
		"key1": "new_value1", // Changed
		"key2": "value2",
	})
	
	changes, err := detector.UpdateSnapshot(newSnap)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	
	if len(changes) != 1 {
		t.Fatalf("expected 1 change, got %d", len(changes))
	}
	
	change := changes[0]
	if change.ChangeType != "ledger_entry" {
		t.Errorf("expected change type 'ledger_entry', got %s", change.ChangeType)
	}
	if change.Key != "key1" {
		t.Errorf("expected key 'key1', got %s", change.Key)
	}
	if change.OldValue != "value1" {
		t.Errorf("expected old value 'value1', got %s", change.OldValue)
	}
	if change.NewValue != "new_value1" {
		t.Errorf("expected new value 'new_value1', got %s", change.NewValue)
	}
	if len(change.AffectedSteps) != 2 {
		t.Errorf("expected 2 affected steps, got %d", len(change.AffectedSteps))
	}
}

func TestUpdateSnapshot_NewEntry(t *testing.T) {
	baseSnap := snapshot.FromMap(map[string]string{
		"key1": "value1",
	})
	
	detector := NewStateChangeDetector(baseSnap)
	
	// Update with new entry
	newSnap := snapshot.FromMap(map[string]string{
		"key1": "value1",
		"key2": "value2", // New
	})
	
	changes, err := detector.UpdateSnapshot(newSnap)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	
	if len(changes) != 1 {
		t.Fatalf("expected 1 change, got %d", len(changes))
	}
	
	change := changes[0]
	if change.Key != "key2" {
		t.Errorf("expected key 'key2', got %s", change.Key)
	}
	if change.OldValue != "" {
		t.Errorf("expected empty old value, got %s", change.OldValue)
	}
	if change.NewValue != "value2" {
		t.Errorf("expected new value 'value2', got %s", change.NewValue)
	}
}

func TestUpdateSnapshot_DeletedEntry(t *testing.T) {
	baseSnap := snapshot.FromMap(map[string]string{
		"key1": "value1",
		"key2": "value2",
	})
	
	detector := NewStateChangeDetector(baseSnap)
	detector.RecordStateDependency(7, "key2")
	
	// Update with deleted entry
	newSnap := snapshot.FromMap(map[string]string{
		"key1": "value1",
		// key2 deleted
	})
	
	changes, err := detector.UpdateSnapshot(newSnap)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	
	if len(changes) != 1 {
		t.Fatalf("expected 1 change, got %d", len(changes))
	}
	
	change := changes[0]
	if change.Key != "key2" {
		t.Errorf("expected key 'key2', got %s", change.Key)
	}
	if change.OldValue != "value2" {
		t.Errorf("expected old value 'value2', got %s", change.OldValue)
	}
	if change.NewValue != "" {
		t.Errorf("expected empty new value, got %s", change.NewValue)
	}
	if len(change.AffectedSteps) != 1 || change.AffectedSteps[0] != 7 {
		t.Errorf("unexpected affected steps: %v", change.AffectedSteps)
	}
}

func TestDetectCodeArtifactChanges(t *testing.T) {
	detector := NewStateChangeDetector(nil)
	
	oldWasm := []byte{0x00, 0x61, 0x73, 0x6d} // WASM magic number
	newWasm := []byte{0x00, 0x61, 0x73, 0x6d, 0x01} // Modified
	
	change, err := detector.DetectCodeArtifactChanges(oldWasm, newWasm, "CONTRACT_ABC")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	
	if change == nil {
		t.Fatal("expected non-nil change")
	}
	if change.ChangeType != "code_artifact" {
		t.Errorf("expected change type 'code_artifact', got %s", change.ChangeType)
	}
	if change.ContractID != "CONTRACT_ABC" {
		t.Errorf("expected contract ID 'CONTRACT_ABC', got %s", change.ContractID)
	}
	if change.OldValue == change.NewValue {
		t.Error("old and new values should differ")
	}
}

func TestDetectCodeArtifactChanges_NoChange(t *testing.T) {
	detector := NewStateChangeDetector(nil)
	
	wasm := []byte{0x00, 0x61, 0x73, 0x6d}
	
	change, err := detector.DetectCodeArtifactChanges(wasm, wasm, "CONTRACT_ABC")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	
	if change != nil {
		t.Error("expected nil change for identical WASM")
	}
}

func TestGetAffectedSteps(t *testing.T) {
	changes := []StateChange{
		{AffectedSteps: []int{1, 3, 5}},
		{AffectedSteps: []int{3, 7, 9}},
		{AffectedSteps: []int{5, 11}},
	}
	
	steps := GetAffectedSteps(changes)
	
	// Should deduplicate: 1, 3, 5, 7, 9, 11
	if len(steps) != 6 {
		t.Errorf("expected 6 unique affected steps, got %d: %v", len(steps), steps)
	}
	
	// Verify all expected steps are present
	stepSet := make(map[int]bool)
	for _, step := range steps {
		stepSet[step] = true
	}
	expected := []int{1, 3, 5, 7, 9, 11}
	for _, exp := range expected {
		if !stepSet[exp] {
			t.Errorf("expected step %d to be in affected steps", exp)
		}
	}
}

func TestComputeRefreshRange(t *testing.T) {
	tests := []struct {
		name          string
		affectedSteps []int
		totalSteps    int
		wantStart     int
		wantEnd       int
	}{
		{
			name:          "no affected steps",
			affectedSteps: []int{},
			totalSteps:    100,
			wantStart:     -1,
			wantEnd:       -1,
		},
		{
			name:          "single affected step",
			affectedSteps: []int{50},
			totalSteps:    100,
			wantStart:     50,
			wantEnd:       99,
		},
		{
			name:          "multiple affected steps",
			affectedSteps: []int{10, 30, 50},
			totalSteps:    100,
			wantStart:     10,
			wantEnd:       99,
		},
		{
			name:          "affected step at start",
			affectedSteps: []int{0, 20},
			totalSteps:    50,
			wantStart:     0,
			wantEnd:       49,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, end := ComputeRefreshRange(tt.affectedSteps, tt.totalSteps)
			if start != tt.wantStart {
				t.Errorf("start step: got %d, want %d", start, tt.wantStart)
			}
			if end != tt.wantEnd {
				t.Errorf("end step: got %d, want %d", end, tt.wantEnd)
			}
		})
	}
}
