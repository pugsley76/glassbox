// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

package trace

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestValidateTraceInputs_Enhanced(t *testing.T) {
	tests := []struct {
		name         string
		verbosity    string
		exportFormat string
		eventFilter  string
		outputPath   string
		wantErr      bool
		errContains  []string
	}{
		{
			name:         "all valid inputs",
			verbosity:    "normal",
			exportFormat: "html",
			eventFilter:  "",
			outputPath:   "./trace.html",
			wantErr:      false,
		},
		{
			name:         "invalid verbosity",
			verbosity:    "ultra",
			exportFormat: "html",
			eventFilter:  "",
			outputPath:   "./trace.html",
			wantErr:      true,
			errContains:  []string{"trace-verbosity", "ultra", "Fix:"},
		},
		{
			name:         "invalid format",
			verbosity:    "normal",
			exportFormat: "yaml",
			eventFilter:  "",
			outputPath:   "./trace.yaml",
			wantErr:      true,
			errContains:  []string{"format", "yaml", "Fix:"},
		},
		{
			name:         "directory path instead of file",
			verbosity:    "normal",
			exportFormat: "html",
			eventFilter:  "",
			outputPath:   "./traces/",
			wantErr:      true,
			errContains:  []string{"directory path", "Fix:", "Example:"},
		},
		{
			name:         "path with traversal",
			verbosity:    "normal",
			exportFormat: "html",
			eventFilter:  "",
			outputPath:   "../../../etc/passwd",
			wantErr:      true,
			errContains:  []string{"traversal", ".."},
		},
		{
			name:         "multiple errors",
			verbosity:    "bad",
			exportFormat: "bad",
			eventFilter:  "",
			outputPath:   "./traces/",
			wantErr:      true,
			errContains:  []string{"trace-verbosity", "format", "directory"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTraceInputs(tt.verbosity, tt.exportFormat, tt.eventFilter, tt.outputPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTraceInputs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				errStr := err.Error()
				for _, want := range tt.errContains {
					if !strings.Contains(errStr, want) {
						t.Errorf("ValidateTraceInputs() error = %v, want it to contain %q", err, want)
					}
				}
			}
		})
	}
}

func TestValidateTraceExportParams(t *testing.T) {
	validTrace := &ExecutionTrace{
		TransactionHash: "abc123",
		StartTime:       time.Now(),
		EndTime:         time.Now().Add(time.Second),
		States: []ExecutionState{
			{Step: 0, Operation: "test"},
		},
	}

	emptyTrace := &ExecutionTrace{
		TransactionHash: "empty",
		StartTime:       time.Now(),
		EndTime:         time.Now(),
		States:          []ExecutionState{},
	}

	tests := []struct {
		name        string
		trace       *ExecutionTrace
		format      string
		outputPath  string
		opts        ExportOptions
		wantErr     bool
		errContains []string
	}{
		{
			name:       "valid params",
			trace:      validTrace,
			format:     "html",
			outputPath: "./output.html",
			opts:       ExportOptions{},
			wantErr:    false,
		},
		{
			name:        "nil trace",
			trace:       nil,
			format:      "html",
			outputPath:  "./output.html",
			opts:        ExportOptions{},
			wantErr:     true,
			errContains: []string{"trace is nil", "Fix:"},
		},
		{
			name:        "empty format",
			trace:       validTrace,
			format:      "",
			outputPath:  "./output.html",
			opts:        ExportOptions{},
			wantErr:     true,
			errContains: []string{"format is empty", "Fix:"},
		},
		{
			name:        "invalid format",
			trace:       validTrace,
			format:      "pdf",
			outputPath:  "./output.pdf",
			opts:        ExportOptions{},
			wantErr:     true,
			errContains: []string{"unsupported", "pdf", "Fix:"},
		},
		{
			name:        "empty output path",
			trace:       validTrace,
			format:      "html",
			outputPath:  "",
			opts:        ExportOptions{},
			wantErr:     true,
			errContains: []string{"output path is empty", "Fix:", "Example:"},
		},
		{
			name:        "directory output path",
			trace:       validTrace,
			format:      "html",
			outputPath:  "./traces/",
			opts:        ExportOptions{},
			wantErr:     true,
			errContains: []string{"directory", "file path"},
		},
		{
			name:        "empty trace states",
			trace:       emptyTrace,
			format:      "html",
			outputPath:  "./output.html",
			opts:        ExportOptions{},
			wantErr:     true,
			errContains: []string{"no steps", "Possible causes:", "Fix:"},
		},
		{
			name:       "too many comments",
			trace:      validTrace,
			format:     "html",
			outputPath: "./output.html",
			opts: ExportOptions{
				Comments: make([]string, 101), // more than max of 100
			},
			wantErr:     true,
			errContains: []string{"too many comments", "101", "Fix:"},
		},
		{
			name:       "comment too long",
			trace:      validTrace,
			format:     "html",
			outputPath: "./output.html",
			opts: ExportOptions{
				Comments: []string{strings.Repeat("x", 10001)}, // more than 10000
			},
			wantErr:     true,
			errContains: []string{"exceeds maximum length", "10001", "Fix:"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTraceExportParams(tt.trace, tt.format, tt.outputPath, tt.opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTraceExportParams() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				errStr := err.Error()
				for _, want := range tt.errContains {
					if !strings.Contains(errStr, want) {
						t.Errorf("ValidateTraceExportParams() error = %v, want it to contain %q", err, want)
					}
				}
			}
		})
	}
}

func TestValidateTraceFormatCompatibility(t *testing.T) {
	validTrace := &ExecutionTrace{
		TransactionHash: "abc123",
		StartTime:       time.Now(),
		EndTime:         time.Now().Add(time.Second),
		States: []ExecutionState{
			{Step: 0, Operation: "test", Arguments: []interface{}{"arg1"}},
			{Step: 1, Operation: "test2", Arguments: []interface{}{"arg2"}},
		},
	}

	corruptedTrace := &ExecutionTrace{
		TransactionHash: "corrupted",
		StartTime:       time.Now(),
		EndTime:         time.Now().Add(time.Second),
		States: []ExecutionState{
			{Step: 0, Operation: "test"},
			{Step: 99, Operation: "test2"}, // Step mismatch!
		},
	}

	hugeArgsTrace := &ExecutionTrace{
		TransactionHash: "huge",
		StartTime:       time.Now(),
		EndTime:         time.Now().Add(time.Second),
		States: []ExecutionState{
			{Step: 0, Operation: "test", Arguments: []interface{}{strings.Repeat("x", 60000)}},
		},
	}

	tests := []struct {
		name        string
		trace       *ExecutionTrace
		format      string
		wantErr     bool
		errContains string
	}{
		{
			name:    "valid json",
			trace:   validTrace,
			format:  "json",
			wantErr: false,
		},
		{
			name:    "valid html",
			trace:   validTrace,
			format:  "html",
			wantErr: false,
		},
		{
			name:    "valid markdown",
			trace:   validTrace,
			format:  "markdown",
			wantErr: false,
		},
		{
			name:    "valid text",
			trace:   validTrace,
			format:  "text",
			wantErr: false,
		},
		{
			name:        "nil trace",
			trace:       nil,
			format:      "html",
			wantErr:     true,
			errContains: "trace is nil",
		},
		{
			name:        "corrupted trace json",
			trace:       corruptedTrace,
			format:      "json",
			wantErr:     true,
			errContains: "step mismatch",
		},
		{
			name:        "huge args html",
			trace:       hugeArgsTrace,
			format:      "html",
			wantErr:     true,
			errContains: "very large arguments",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTraceFormatCompatibility(tt.trace, tt.format)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTraceFormatCompatibility() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil {
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("ValidateTraceFormatCompatibility() error = %v, want it to contain %q", err, tt.errContains)
				}
			}
		})
	}
}

func TestValidateExecutionTrace_Enhanced(t *testing.T) {
	validTrace := &ExecutionTrace{
		TransactionHash: "abc123",
		States: []ExecutionState{
			{Step: 0, Operation: "test", EventType: "contract_call"},
			{Step: 1, Operation: "test2", EventType: "diagnostic"},
		},
	}

	emptyTrace := &ExecutionTrace{
		TransactionHash: "empty",
		States:          []ExecutionState{},
	}

	mismatchTrace := &ExecutionTrace{
		TransactionHash: "mismatch",
		States: []ExecutionState{
			{Step: 0, Operation: "test"},
			{Step: 5, Operation: "test2"}, // Wrong step number
		},
	}

	tests := []struct {
		name            string
		trace           *ExecutionTrace
		wantIssueCount  int
		issuesContain   []string
	}{
		{
			name:           "valid trace",
			trace:          validTrace,
			wantIssueCount: 0,
		},
		{
			name:           "nil trace",
			trace:          nil,
			wantIssueCount: 1,
			issuesContain:  []string{"trace is nil"},
		},
		{
			name:           "empty trace",
			trace:          emptyTrace,
			wantIssueCount: 1,
			issuesContain:  []string{"no steps"},
		},
		{
			name:           "step mismatch",
			trace:          mismatchTrace,
			wantIssueCount: 1,
			issuesContain:  []string{"step index mismatch", "position 1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issues := ValidateExecutionTrace(tt.trace)
			if len(issues) != tt.wantIssueCount {
				t.Errorf("ValidateExecutionTrace() returned %d issues, want %d. Issues: %v", 
					len(issues), tt.wantIssueCount, issues)
			}
			for _, want := range tt.issuesContain {
				found := false
				for _, issue := range issues {
					if strings.Contains(issue, want) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("ValidateExecutionTrace() issues %v should contain %q", issues, want)
				}
			}
		})
	}
}

func TestTraceInputError_Formatting(t *testing.T) {
	tests := []struct {
		name     string
		failures []string
		want     string
	}{
		{
			name:     "single failure",
			failures: []string{"invalid format"},
			want:     "invalid format",
		},
		{
			name:     "multiple failures",
			failures: []string{"error 1", "error 2", "error 3"},
			want:     "3 trace input validation error(s):\n  1. error 1\n  2. error 2\n  3. error 3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &TraceInputError{Failures: tt.failures}
			if got := err.Error(); got != tt.want {
				t.Errorf("TraceInputError.Error() = %v, want %v", got, tt.want)
			}
		})
	}
}

// ── Source mapping validation ────────────────────────────────────────────────

func TestValidateSourceMapping_UnknownQuality_HasActionableDiagnostic(t *testing.T) {
	// When source mapping fails to resolve, the trace validation should surface
	// a diagnostic that includes the contract hash and actionable hints.
	tr := NewExecutionTrace("deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef", 0)
	tr.AddState(ExecutionState{
		Operation:  "contract_call",
		ContractID: "CDLZFC3SYJYDZT7K67VZ75HPJVIEUVNIXF47ZG2FB2RMQQVU2HHGCYSC",
		Function:   "transfer",
		Timestamp:  time.Now(),
	})

	issues := ValidateExecutionTrace(tr)
	found := false
	for _, issue := range issues {
		if strings.Contains(issue, "source") || strings.Contains(issue, "DWARF") || strings.Contains(issue, "contract") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected at least one issue mentioning source/DWARF/contract for trace with unresolved source mapping, got: %v", issues)
	}
}

func TestSplitPane_NoSource_ContainsContractSourceHint(t *testing.T) {
	node := NewTraceNode("call-1", "contract_call")
	node.ContractID = "CDLZFC3SYJYDZT7K67VZ75HPJVIEUVNIXF47ZG2FB2RMQQVU2HHGCYSC"
	node.Function = "transfer"

	var buf bytes.Buffer
	pane := &SplitPane{Width: 80, TraceRows: 6, SrcRows: 8}
	pane.Render(&buf, node, nil)

	out := buf.String()
	if !strings.Contains(out, "--contract-source") {
		t.Errorf("split pane no-source output should suggest --contract-source, got:\n%s", out)
	}
	if !strings.Contains(out, "--skip-source-mapping") {
		t.Errorf("split pane no-source output should suggest --skip-source-mapping, got:\n%s", out)
	}
	if !strings.Contains(out, "debug = true") {
		t.Errorf("split pane no-source output should suggest recompiling with debug=true, got:\n%s", out)
	}
}

func TestValidateSourceMapping_ValidTrace_NoSourceWarning(t *testing.T) {
	// A valid trace with proper steps should not raise source-mapping warnings
	// when validation is run independently.
	tr := NewExecutionTrace("abc123", 0)
	for i := 0; i < 3; i++ {
		tr.AddState(ExecutionState{
			Operation:  "contract_call",
			ContractID: "CTEST",
			Function:   "mint",
			Timestamp:  time.Now(),
		})
	}

	issues := ValidateExecutionTrace(tr)
	var sourceWarnings []string
	for _, issue := range issues {
		if strings.Contains(strings.ToLower(issue), "source") && strings.Contains(strings.ToLower(issue), "mapping") {
			sourceWarnings = append(sourceWarnings, issue)
		}
	}
	if len(sourceWarnings) > 0 {
		t.Errorf("valid trace should not produce source mapping warnings, got: %v", sourceWarnings)
	}
}
