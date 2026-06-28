// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

package trace

import (
	"strings"
	"testing"
	"time"

	"github.com/dotandev/glassbox/internal/simulator"
)

// makeTestDiagEvent returns a minimal simulator.DiagnosticEvent for test use.
func makeTestDiagEvent(eventType string) simulator.DiagnosticEvent {
	return simulator.DiagnosticEvent{EventType: eventType, Topics: []string{}}
}

// ---------------------------------------------------------------------------
// ValidateExecutionTrace — extended accuracy/context checks
// ---------------------------------------------------------------------------

func TestValidateExecutionTrace_ZeroTimestamp(t *testing.T) {
	tr := NewExecutionTrace("tx-zero-ts", 0)
	tr.States = []ExecutionState{
		{Step: 0, Operation: "call", EventType: "contract_call"},
		// step 1: timestamp intentionally left zero
		{Step: 1, Operation: "return", EventType: "return"},
	}
	issues := ValidateExecutionTrace(tr)
	found := false
	for _, iss := range issues {
		if strings.Contains(iss, "timestamp is zero") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected zero-timestamp issue, got: %v", issues)
	}
}

func TestValidateExecutionTrace_NonMonotonicTimestamps(t *testing.T) {
	now := time.Now()
	tr := NewExecutionTrace("tx-nonmono", 0)
	tr.States = []ExecutionState{
		{Step: 0, Operation: "call", Timestamp: now},
		{Step: 1, Operation: "ret", Timestamp: now.Add(-1 * time.Second)}, // goes backwards
	}
	issues := ValidateExecutionTrace(tr)
	found := false
	for _, iss := range issues {
		if strings.Contains(iss, "non-monotonic") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected non-monotonic timestamp issue, got: %v", issues)
	}
}

func TestValidateExecutionTrace_MissingOperationAndEventType(t *testing.T) {
	now := time.Now()
	tr := NewExecutionTrace("tx-nocontext", 0)
	tr.States = []ExecutionState{
		{Step: 0, Timestamp: now}, // neither Operation nor EventType
	}
	issues := ValidateExecutionTrace(tr)
	found := false
	for _, iss := range issues {
		if strings.Contains(iss, "no context") || strings.Contains(iss, "neither Operation") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected missing context issue, got: %v", issues)
	}
}

func TestValidateExecutionTrace_ContractIDWithoutFunction(t *testing.T) {
	now := time.Now()
	tr := NewExecutionTrace("tx-nofunc", 0)
	tr.States = []ExecutionState{
		{Step: 0, Operation: "call", EventType: "contract_call",
			ContractID: "CABC123", Function: "", Timestamp: now},
	}
	issues := ValidateExecutionTrace(tr)
	found := false
	for _, iss := range issues {
		if strings.Contains(iss, "Function is empty") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected missing Function issue, got: %v", issues)
	}
}

func TestValidateExecutionTrace_DiagnosticEventsMismatch(t *testing.T) {
	now := time.Now()
	tr := NewExecutionTrace("tx-evtmismatch", 0)
	tr.States = []ExecutionState{
		{Step: 0, Operation: "call", Timestamp: now},
		{Step: 1, Operation: "ret", Timestamp: now.Add(time.Millisecond)},
	}
	// Mismatched: 1 diagnostic event for 2 states
	tr.DiagnosticEvents = tr.DiagnosticEvents[:0]
	// Append a single placeholder event directly via slice literal
	tr.DiagnosticEvents = append(tr.DiagnosticEvents,
		makeTestDiagEvent("contract_call"),
	)
	issues := ValidateExecutionTrace(tr)
	found := false
	for _, iss := range issues {
		if strings.Contains(iss, "diagnostic event count") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected diagnostic event count mismatch issue, got: %v", issues)
	}
}

func TestValidateExecutionTrace_FullyValid(t *testing.T) {
	now := time.Now()
	tr := NewExecutionTrace("tx-valid", 0)
	tr.States = []ExecutionState{
		{Step: 0, Operation: "call", EventType: "contract_call",
			ContractID: "CABC", Function: "transfer", Timestamp: now},
		{Step: 1, Operation: "return", EventType: "return",
			Timestamp: now.Add(time.Millisecond)},
	}
	issues := ValidateExecutionTrace(tr)
	if len(issues) != 0 {
		t.Errorf("fully valid trace should have no issues, got: %v", issues)
	}
}

// ---------------------------------------------------------------------------
// ValidateTraceAccuracy
// ---------------------------------------------------------------------------

func TestValidateTraceAccuracy_Nil(t *testing.T) {
	err := ValidateTraceAccuracy(nil)
	if err == nil {
		t.Fatal("nil trace should produce error")
	}
	if !strings.Contains(err.Error(), "nil") {
		t.Errorf("error should mention nil, got: %v", err)
	}
}

func TestValidateTraceAccuracy_Clean(t *testing.T) {
	now := time.Now()
	tr := NewExecutionTrace("tx-clean", 0)
	tr.States = []ExecutionState{
		{Step: 0, Operation: "call", EventType: "contract_call",
			ContractID: "CABC", Function: "xfer", Timestamp: now},
	}
	if err := ValidateTraceAccuracy(tr); err != nil {
		t.Errorf("clean trace should pass accuracy check, got: %v", err)
	}
}

func TestValidateTraceAccuracy_HardIssue_IndexMismatch(t *testing.T) {
	now := time.Now()
	tr := NewExecutionTrace("tx-hard", 0)
	tr.States = []ExecutionState{
		{Step: 0, Operation: "call", Timestamp: now},
		{Step: 99, Operation: "ret", Timestamp: now.Add(time.Millisecond)}, // wrong index
	}
	err := ValidateTraceAccuracy(tr)
	if err == nil {
		t.Fatal("step index mismatch should produce accuracy error")
	}
	if !strings.Contains(err.Error(), "[accuracy]") {
		t.Errorf("hard issues should be tagged [accuracy], got: %v", err)
	}
}

func TestValidateTraceAccuracy_SoftIssue_MissingContext(t *testing.T) {
	now := time.Now()
	tr := NewExecutionTrace("tx-soft", 0)
	tr.States = []ExecutionState{
		// No Operation or EventType → soft context warning
		{Step: 0, Timestamp: now},
	}
	err := ValidateTraceAccuracy(tr)
	if err == nil {
		t.Fatal("missing context should produce accuracy warning")
	}
	if !strings.Contains(err.Error(), "[context]") {
		t.Errorf("soft issues should be tagged [context], got: %v", err)
	}
}

func TestValidateTraceAccuracy_MultipleIssuesSurfacedTogether(t *testing.T) {
	now := time.Now()
	tr := NewExecutionTrace("tx-multi", 0)
	tr.States = []ExecutionState{
		{Step: 0, Timestamp: now},                                          // missing context (soft)
		{Step: 5, Timestamp: now.Add(time.Millisecond)},                    // index mismatch (hard)
	}
	err := ValidateTraceAccuracy(tr)
	if err == nil {
		t.Fatal("expected errors for multiple issues")
	}
	msg := err.Error()
	if !strings.Contains(msg, "[accuracy]") {
		t.Errorf("expected [accuracy] tag, got: %v", msg)
	}
	if !strings.Contains(msg, "[context]") {
		t.Errorf("expected [context] tag, got: %v", msg)
	}
}
