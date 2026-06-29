// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/dotandev/glassbox/internal/clioutput"
	"github.com/dotandev/glassbox/internal/protocolreg"
)

// ── --format flag validation ──────────────────────────────────────────────────

// TestProtocolDiagnoseFormat_ValidValues verifies that "text", "json", and ""
// are the only accepted --format values.
func TestProtocolDiagnoseFormat_ValidValues(t *testing.T) {
	validFormats := []string{"text", "json", "TEXT", "JSON", ""}
	for _, f := range validFormats {
		normalized := strings.ToLower(strings.TrimSpace(f))
		if normalized != "" && normalized != "text" && normalized != "json" {
			t.Errorf("format %q should be valid but was rejected", f)
		}
	}
}

func TestProtocolDiagnoseFormat_InvalidValues(t *testing.T) {
	invalidFormats := []string{"yaml", "xml", "csv", "pretty", "raw", "table"}
	for _, f := range invalidFormats {
		normalized := strings.ToLower(strings.TrimSpace(f))
		if normalized == "" || normalized == "text" || normalized == "json" {
			t.Errorf("format %q should be invalid but was accepted", f)
		}
	}
}

// TestProtocolDiagnoseFormat_JSONProducesValidEnvelope verifies that selecting
// JSON output wraps the DiagnosticReport in a valid clioutput.Envelope.
func TestProtocolDiagnoseFormat_JSONProducesValidEnvelope(t *testing.T) {
	report := &protocolreg.DiagnosticReport{
		Platform: "linux",
		Scheme:   protocolreg.Scheme,
		Status:   protocolreg.StatusOK,
		Checks:   []string{"Desktop file found"},
	}

	var buf bytes.Buffer
	if err := clioutput.Write(&buf, "protocol:diagnose", report); err != nil {
		t.Fatalf("clioutput.Write: %v", err)
	}

	var env clioutput.Envelope
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("unmarshal JSON envelope: %v", err)
	}

	if env.SchemaVersion != clioutput.SchemaVersion {
		t.Errorf("schema_version = %q, want %q", env.SchemaVersion, clioutput.SchemaVersion)
	}
	if env.Command != "protocol:diagnose" {
		t.Errorf("command = %q, want 'protocol:diagnose'", env.Command)
	}
	if env.GlassboxVersion == "" {
		t.Error("glassbox_version must not be empty")
	}
	if env.GeneratedAt.IsZero() {
		t.Error("generated_at must be populated")
	}
}

// TestProtocolDiagnoseFormat_JSONEnvelope_ContainsDiagData verifies that the
// JSON envelope's data field contains the actual diagnostic payload.
func TestProtocolDiagnoseFormat_JSONEnvelope_ContainsDiagData(t *testing.T) {
	report := &protocolreg.DiagnosticReport{
		Platform: "testplatform",
		Scheme:   protocolreg.Scheme,
		Status:   protocolreg.StatusNotRegistered,
		Issues:   []string{"desktop file missing"},
	}

	var buf bytes.Buffer
	if err := clioutput.Write(&buf, "protocol:diagnose", report); err != nil {
		t.Fatalf("clioutput.Write: %v", err)
	}

	raw := buf.String()
	if !strings.Contains(raw, "not_registered") {
		t.Errorf("JSON output should include the status value 'not_registered'; got: %s", raw)
	}
	if !strings.Contains(raw, "desktop file missing") {
		t.Errorf("JSON output should include the issue text; got: %s", raw)
	}
	if !strings.Contains(raw, "testplatform") {
		t.Errorf("JSON output should include the platform field; got: %s", raw)
	}
}

// ── WantsJSON helper ──────────────────────────────────────────────────────────

func TestWantsJSON_JSONFlagTrue(t *testing.T) {
	if !clioutput.WantsJSON(true, "") {
		t.Error("WantsJSON should return true when --json flag is set")
	}
}

func TestWantsJSON_FormatJSON(t *testing.T) {
	if !clioutput.WantsJSON(false, "json") {
		t.Error("WantsJSON should return true for --format json")
	}
}

func TestWantsJSON_FormatText(t *testing.T) {
	if clioutput.WantsJSON(false, "text") {
		t.Error("WantsJSON should return false for --format text")
	}
}

func TestWantsJSON_NoFlags(t *testing.T) {
	if clioutput.WantsJSON(false, "") {
		t.Error("WantsJSON should return false when no flags are set")
	}
}

// ── DiagnosticReport JSON field coverage ─────────────────────────────────────

// TestDiagnosticReport_JSONFieldNames verifies that the DiagnosticReport fields
// serialise to the expected JSON keys (catches accidental tag renames).
func TestDiagnosticReport_JSONFieldNames(t *testing.T) {
	report := &protocolreg.DiagnosticReport{
		Platform:           "linux",
		Scheme:             "glassbox",
		Status:             protocolreg.StatusDegraded,
		Checks:             []string{"one"},
		Issues:             []string{"two"},
		RemediationSteps:   []string{"three"},
		ExecutablePath:     "/usr/bin/glassbox",
		RegisteredHandler:  "/usr/bin/other",
		HandlerMatchesSelf: false,
		ConflictDetected:   true,
		ConflictingHandler: "/usr/bin/other",
	}

	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	out := string(raw)

	expectedKeys := []string{
		`"platform"`, `"scheme"`, `"status"`, `"checks"`, `"issues"`,
		`"remediation_steps"`, `"executable_path"`, `"registered_handler"`,
		`"handler_matches_self"`, `"conflict_detected"`, `"conflicting_handler"`,
	}
	for _, key := range expectedKeys {
		if !strings.Contains(out, key) {
			t.Errorf("JSON output missing field %s; got: %s", key, out)
		}
	}
}
