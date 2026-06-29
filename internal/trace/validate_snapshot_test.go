// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

package trace

import (
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// ValidateSnapshotForExport
// ---------------------------------------------------------------------------

func TestValidateSnapshotForExport_OOMError(t *testing.T) {
	err := ValidateSnapshotForExport(0, 100, true)
	if err == nil {
		t.Fatal("expected error for OOM condition")
	}
	msg := err.Error()
	if !strings.Contains(msg, "memory pressure") {
		t.Errorf("error should mention memory pressure, got: %v", msg)
	}
	if !strings.Contains(msg, "Fix:") {
		t.Errorf("error should include a Fix hint, got: %v", msg)
	}
}

func TestValidateSnapshotForExport_NoSnapshotsWithSteps(t *testing.T) {
	err := ValidateSnapshotForExport(0, 50, false)
	if err == nil {
		t.Fatal("expected error when no snapshots captured but steps executed")
	}
	msg := err.Error()
	if !strings.Contains(msg, "no snapshots were captured") {
		t.Errorf("error should mention no snapshots captured, got: %v", msg)
	}
	if !strings.Contains(msg, "50 steps") {
		t.Errorf("error should include step count, got: %v", msg)
	}
	if !strings.Contains(msg, "Fix:") {
		t.Errorf("error should include a Fix hint, got: %v", msg)
	}
}

func TestValidateSnapshotForExport_SparseSnapshotCoverage(t *testing.T) {
	// 1 snapshot for 1000 steps → below 1-per-200 threshold (expect at least 5)
	err := ValidateSnapshotForExport(1, 1000, false)
	if err == nil {
		t.Fatal("expected error for sparse snapshot coverage")
	}
	msg := err.Error()
	if !strings.Contains(msg, "sparse snapshot coverage") {
		t.Errorf("error should mention sparse coverage, got: %v", msg)
	}
	if !strings.Contains(msg, "Fix:") {
		t.Errorf("error should include a Fix hint, got: %v", msg)
	}
}

func TestValidateSnapshotForExport_AdequateCoverage(t *testing.T) {
	// 10 snapshots for 1000 steps → above threshold (need 5), should pass
	err := ValidateSnapshotForExport(10, 1000, false)
	if err != nil {
		t.Errorf("adequate snapshot coverage should not error, got: %v", err)
	}
}

func TestValidateSnapshotForExport_ZeroSteps(t *testing.T) {
	// 0 steps with 0 snapshots is valid (nothing ran, nothing to snapshot)
	err := ValidateSnapshotForExport(0, 0, false)
	if err != nil {
		t.Errorf("zero steps with zero snapshots should not error, got: %v", err)
	}
}

func TestValidateSnapshotForExport_ExactlyAtThreshold(t *testing.T) {
	// Exactly at threshold: 5 snapshots for 1000 steps (1000/200 = 5), passes
	err := ValidateSnapshotForExport(5, 1000, false)
	if err != nil {
		t.Errorf("snapshot count at exact threshold should not error, got: %v", err)
	}
}

func TestValidateSnapshotForExport_OOMTakesPriority(t *testing.T) {
	// OOM should be reported even if snapshot count looks OK
	err := ValidateSnapshotForExport(100, 1000, true)
	if err == nil {
		t.Fatal("OOM should always produce an error")
	}
	if !strings.Contains(err.Error(), "memory pressure") {
		t.Errorf("OOM error should mention memory pressure, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// ValidateJSONSchemaVersion
// ---------------------------------------------------------------------------

func TestValidateJSONSchemaVersion_Valid(t *testing.T) {
	if err := ValidateJSONSchemaVersion("1.0"); err != nil {
		t.Errorf("valid schema version should pass, got: %v", err)
	}
}

func TestValidateJSONSchemaVersion_Empty(t *testing.T) {
	err := ValidateJSONSchemaVersion("")
	if err == nil {
		t.Fatal("empty version should fail")
	}
	if !strings.Contains(err.Error(), "schema_version is empty") {
		t.Errorf("error should mention empty version, got: %v", err)
	}
}

func TestValidateJSONSchemaVersion_Whitespace(t *testing.T) {
	err := ValidateJSONSchemaVersion("   ")
	if err == nil {
		t.Fatal("whitespace-only version should fail")
	}
	if !strings.Contains(err.Error(), "schema_version is empty") {
		t.Errorf("error should mention empty version, got: %v", err)
	}
}

func TestValidateJSONSchemaVersion_WrongFormat(t *testing.T) {
	cases := []string{"1", "1.0.0", "v1.0", "1-0", ""}
	for _, c := range cases {
		if c == "" {
			continue // already tested above
		}
		err := ValidateJSONSchemaVersion(c)
		if err == nil {
			t.Errorf("invalid version %q should fail", c)
		}
	}
}

func TestValidateJSONSchemaVersion_NonNumeric(t *testing.T) {
	err := ValidateJSONSchemaVersion("a.b")
	if err == nil {
		t.Fatal("non-numeric version should fail")
	}
	if !strings.Contains(err.Error(), "non-numeric") {
		t.Errorf("error should mention non-numeric characters, got: %v", err)
	}
}

func TestValidateJSONSchemaVersion_Unsupported(t *testing.T) {
	err := ValidateJSONSchemaVersion("99.0")
	if err == nil {
		t.Fatal("unsupported version should fail")
	}
	msg := err.Error()
	if !strings.Contains(msg, "not supported") {
		t.Errorf("error should mention unsupported version, got: %v", msg)
	}
	if !strings.Contains(msg, "Fix:") {
		t.Errorf("error should include a Fix hint, got: %v", msg)
	}
}

// ---------------------------------------------------------------------------
// ValidateExecutionTrace
// ---------------------------------------------------------------------------

func TestValidateExecutionTrace_Nil(t *testing.T) {
	issues := ValidateExecutionTrace(nil)
	if len(issues) == 0 {
		t.Fatal("nil trace should produce issues")
	}
	if !strings.Contains(issues[0], "nil") {
		t.Errorf("issue should mention nil, got: %v", issues[0])
	}
}

func TestValidateExecutionTrace_Empty(t *testing.T) {
	tr := NewExecutionTrace("test-tx", 0)
	issues := ValidateExecutionTrace(tr)
	if len(issues) == 0 {
		t.Fatal("empty trace should produce issues")
	}
	if !strings.Contains(issues[0], "no steps") {
		t.Errorf("issue should mention no steps, got: %v", issues[0])
	}
}

func TestValidateExecutionTrace_StepIndexMismatch(t *testing.T) {
	tr := NewExecutionTrace("test-tx", 0)
	tr.States = []ExecutionState{
		{Step: 0, Operation: "first"},
		{Step: 5, Operation: "second"}, // wrong index
	}
	issues := ValidateExecutionTrace(tr)
	if len(issues) == 0 {
		t.Fatal("step index mismatch should produce issues")
	}
	found := false
	for _, issue := range issues {
		if strings.Contains(issue, "mismatch") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("issues should mention mismatch, got: %v", issues)
	}
}

func TestValidateExecutionTrace_Valid(t *testing.T) {
	tr := NewExecutionTrace("test-tx", 0)
	tr.States = []ExecutionState{
		{Step: 0, Operation: "call", EventType: "contract_call"},
		{Step: 1, Operation: "return", EventType: "return"},
	}
	issues := ValidateExecutionTrace(tr)
	if len(issues) != 0 {
		t.Errorf("valid trace should produce no issues, got: %v", issues)
	}
}

func TestValidateExecutionTrace_UnrecognisedEventType(t *testing.T) {
	tr := NewExecutionTrace("test-tx", 0)
	tr.States = []ExecutionState{
		{Step: 0, Operation: "call", EventType: "totally_unknown_event_xyz"},
	}
	issues := ValidateExecutionTrace(tr)
	if len(issues) == 0 {
		t.Fatal("unrecognised event type should produce issues")
	}
	if !strings.Contains(issues[0], "unrecognised event type") {
		t.Errorf("issue should mention unrecognised event type, got: %v", issues[0])
	}
}

// ---------------------------------------------------------------------------
// ValidateTraceInputs
// ---------------------------------------------------------------------------

func TestValidateTraceInputs_AllValid(t *testing.T) {
	err := ValidateTraceInputs("normal", "html", "", "./out/trace.html")
	if err != nil {
		t.Errorf("all-valid inputs should pass, got: %v", err)
	}
}

func TestValidateTraceInputs_InvalidVerbosity(t *testing.T) {
	err := ValidateTraceInputs("ultra", "html", "", "./out/trace.html")
	if err == nil {
		t.Fatal("invalid verbosity should fail")
	}
	if !strings.Contains(err.Error(), "--trace-verbosity") {
		t.Errorf("error should mention --trace-verbosity, got: %v", err)
	}
}

func TestValidateTraceInputs_InvalidFormat(t *testing.T) {
	err := ValidateTraceInputs("", "yaml", "", "./out/trace.yaml")
	if err == nil {
		t.Fatal("invalid format should fail")
	}
	if !strings.Contains(err.Error(), "trace export format") {
		t.Errorf("error should mention trace export format, got: %v", err)
	}
}

func TestValidateTraceInputs_DirectoryPath(t *testing.T) {
	err := ValidateTraceInputs("", "html", "", "./output/")
	if err == nil {
		t.Fatal("directory path should fail")
	}
	if !strings.Contains(err.Error(), "directory path") {
		t.Errorf("error should mention directory path, got: %v", err)
	}
}

func TestValidateTraceInputs_TraversalPath(t *testing.T) {
	err := ValidateTraceInputs("", "html", "", "../../../etc/passwd")
	if err == nil {
		t.Fatal("path traversal should fail")
	}
	if !strings.Contains(err.Error(), "traversal") {
		t.Errorf("error should mention traversal, got: %v", err)
	}
}

func TestValidateTraceInputs_MultipleErrors(t *testing.T) {
	// Both invalid verbosity and invalid format → both errors reported
	err := ValidateTraceInputs("ultra", "yaml", "", "./output/")
	if err == nil {
		t.Fatal("multiple invalid inputs should fail")
	}
	msg := err.Error()
	if !strings.Contains(msg, "validation error") {
		t.Errorf("error should mention validation errors (plural), got: %v", msg)
	}
}

// ---------------------------------------------------------------------------
// TraceInputError formatting
// ---------------------------------------------------------------------------

func TestTraceInputError_SingleFailure(t *testing.T) {
	e := &TraceInputError{Failures: []string{"something went wrong"}}
	if e.Error() != "something went wrong" {
		t.Errorf("single failure should return bare message, got: %q", e.Error())
	}
}

func TestTraceInputError_MultipleFailures(t *testing.T) {
	e := &TraceInputError{Failures: []string{"error one", "error two", "error three"}}
	msg := e.Error()
	if !strings.Contains(msg, "3 trace input validation error(s)") {
		t.Errorf("multiple failures should show count, got: %q", msg)
	}
	if !strings.Contains(msg, "1. error one") {
		t.Errorf("should enumerate failures, got: %q", msg)
	}
}
