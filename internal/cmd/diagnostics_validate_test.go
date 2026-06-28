// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"strings"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// ValidateDiagnosticsOutput
// ---------------------------------------------------------------------------

func TestValidateDiagnosticsOutput_Nil(t *testing.T) {
	issues := ValidateDiagnosticsOutput(nil)
	if len(issues) == 0 {
		t.Fatal("nil output should produce issues")
	}
	if !strings.Contains(issues[0], "nil") {
		t.Errorf("issue should mention nil, got: %v", issues[0])
	}
}

func TestValidateDiagnosticsOutput_Healthy(t *testing.T) {
	out := &DiagnosticsOutput{
		OverallHealth: "Healthy",
		RPC: []RPCStatus{
			{URL: "https://rpc.example.com", Status: "[OK]", Healthy: true},
		},
		Cache:  CacheStatus{Healthy: true},
		Config: ConfigSummary{Healthy: true},
	}
	if issues := ValidateDiagnosticsOutput(out); len(issues) != 0 {
		t.Errorf("healthy output should have no issues, got: %v", issues)
	}
}

func TestValidateDiagnosticsOutput_EmptyOverallHealth(t *testing.T) {
	out := &DiagnosticsOutput{
		OverallHealth: "",
		Cache:         CacheStatus{Healthy: true},
		Config:        ConfigSummary{Healthy: true},
	}
	issues := ValidateDiagnosticsOutput(out)
	found := false
	for _, iss := range issues {
		if strings.Contains(iss, "OverallHealth is empty") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected empty OverallHealth issue, got: %v", issues)
	}
}

func TestValidateDiagnosticsOutput_UnknownOverallHealth(t *testing.T) {
	out := &DiagnosticsOutput{
		OverallHealth: "Unknown",
		Cache:         CacheStatus{Healthy: true},
		Config:        ConfigSummary{Healthy: true},
	}
	issues := ValidateDiagnosticsOutput(out)
	found := false
	for _, iss := range issues {
		if strings.Contains(iss, "unexpected value") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected unexpected-value issue, got: %v", issues)
	}
}

func TestValidateDiagnosticsOutput_RPCEmptyStatus(t *testing.T) {
	out := &DiagnosticsOutput{
		OverallHealth: "Healthy",
		RPC: []RPCStatus{
			{URL: "https://rpc.example.com", Status: "", Healthy: true},
		},
		Cache:  CacheStatus{Healthy: true},
		Config: ConfigSummary{Healthy: true},
	}
	issues := ValidateDiagnosticsOutput(out)
	found := false
	for _, iss := range issues {
		if strings.Contains(iss, "Status is empty") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected empty-status issue, got: %v", issues)
	}
}

func TestValidateDiagnosticsOutput_UnhealthyRPCMissingError(t *testing.T) {
	out := &DiagnosticsOutput{
		OverallHealth: "Degraded",
		RPC: []RPCStatus{
			{URL: "https://rpc.example.com", Status: "[FAIL]", Healthy: false, Error: ""},
		},
		Cache:  CacheStatus{Healthy: true},
		Config: ConfigSummary{Healthy: true},
	}
	issues := ValidateDiagnosticsOutput(out)
	found := false
	for _, iss := range issues {
		if strings.Contains(iss, "Error is empty") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected missing-error issue, got: %v", issues)
	}
}

func TestValidateDiagnosticsOutput_InconsistentStatusOKButUnhealthy(t *testing.T) {
	out := &DiagnosticsOutput{
		OverallHealth: "Degraded",
		RPC: []RPCStatus{
			{URL: "https://rpc.example.com", Status: "[OK]", Healthy: false, Error: "timeout"},
		},
		Cache:  CacheStatus{Healthy: true},
		Config: ConfigSummary{Healthy: true},
	}
	issues := ValidateDiagnosticsOutput(out)
	found := false
	for _, iss := range issues {
		if strings.Contains(iss, "inconsistent") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected inconsistent status/healthy issue, got: %v", issues)
	}
}

func TestValidateDiagnosticsOutput_OverallHealthyButRPCUnhealthy(t *testing.T) {
	out := &DiagnosticsOutput{
		OverallHealth: "Healthy",
		RPC: []RPCStatus{
			{URL: "https://rpc.example.com", Status: "[FAIL]", Healthy: false, Error: "refused"},
		},
		Cache:  CacheStatus{Healthy: true},
		Config: ConfigSummary{Healthy: true},
	}
	issues := ValidateDiagnosticsOutput(out)
	found := false
	for _, iss := range issues {
		if strings.Contains(iss, "misleading") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected misleading-overall-health issue, got: %v", issues)
	}
}

func TestValidateDiagnosticsOutput_OverallHealthyButCacheUnhealthy(t *testing.T) {
	out := &DiagnosticsOutput{
		OverallHealth: "Healthy",
		Cache:         CacheStatus{Healthy: false},
		Config:        ConfigSummary{Healthy: true},
	}
	issues := ValidateDiagnosticsOutput(out)
	found := false
	for _, iss := range issues {
		if strings.Contains(iss, "cache is unhealthy") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected cache-unhealthy issue, got: %v", issues)
	}
}

func TestValidateDiagnosticsOutput_OverallHealthyButConfigUnhealthy(t *testing.T) {
	out := &DiagnosticsOutput{
		OverallHealth: "Healthy",
		Cache:         CacheStatus{Healthy: true},
		Config:        ConfigSummary{Healthy: false},
	}
	issues := ValidateDiagnosticsOutput(out)
	found := false
	for _, iss := range issues {
		if strings.Contains(iss, "config is unhealthy") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected config-unhealthy issue, got: %v", issues)
	}
}

func TestValidateDiagnosticsOutput_MultipleIssues(t *testing.T) {
	out := &DiagnosticsOutput{
		OverallHealth: "Healthy",
		RPC: []RPCStatus{
			{URL: "https://bad.example.com", Status: "[FAIL]", Healthy: false, Error: ""},
		},
		Cache:  CacheStatus{Healthy: false},
		Config: ConfigSummary{Healthy: false},
	}
	issues := ValidateDiagnosticsOutput(out)
	if len(issues) < 3 {
		t.Errorf("expected at least 3 issues, got %d: %v", len(issues), issues)
	}
}

// ---------------------------------------------------------------------------
// computeOverallHealth
// ---------------------------------------------------------------------------

func TestComputeOverallHealth_Nil(t *testing.T) {
	if got := computeOverallHealth(nil); got != "Degraded" {
		t.Errorf("nil should produce Degraded, got %q", got)
	}
}

func TestComputeOverallHealth_AllHealthy(t *testing.T) {
	out := &DiagnosticsOutput{
		RPC:    []RPCStatus{{Healthy: true}},
		Cache:  CacheStatus{Healthy: true},
		Config: ConfigSummary{Healthy: true},
	}
	if got := computeOverallHealth(out); got != "Healthy" {
		t.Errorf("all-healthy should produce Healthy, got %q", got)
	}
}

func TestComputeOverallHealth_UnhealthyRPC(t *testing.T) {
	out := &DiagnosticsOutput{
		RPC:    []RPCStatus{{Healthy: false}},
		Cache:  CacheStatus{Healthy: true},
		Config: ConfigSummary{Healthy: true},
	}
	if got := computeOverallHealth(out); got != "Degraded" {
		t.Errorf("unhealthy RPC should produce Degraded, got %q", got)
	}
}

func TestComputeOverallHealth_UnhealthyCache(t *testing.T) {
	out := &DiagnosticsOutput{
		RPC:    []RPCStatus{{Healthy: true}},
		Cache:  CacheStatus{Healthy: false},
		Config: ConfigSummary{Healthy: true},
	}
	if got := computeOverallHealth(out); got != "Degraded" {
		t.Errorf("unhealthy cache should produce Degraded, got %q", got)
	}
}

func TestComputeOverallHealth_UnhealthyConfig(t *testing.T) {
	out := &DiagnosticsOutput{
		RPC:    []RPCStatus{{Healthy: true}},
		Cache:  CacheStatus{Healthy: true},
		Config: ConfigSummary{Healthy: false},
	}
	if got := computeOverallHealth(out); got != "Degraded" {
		t.Errorf("unhealthy config should produce Degraded, got %q", got)
	}
}

func TestComputeOverallHealth_NoRPCEndpoints(t *testing.T) {
	out := &DiagnosticsOutput{
		RPC:    []RPCStatus{},
		Cache:  CacheStatus{Healthy: true},
		Config: ConfigSummary{Healthy: true},
	}
	if got := computeOverallHealth(out); got != "Healthy" {
		t.Errorf("no RPC endpoints with healthy cache/config should be Healthy, got %q", got)
	}
}

// ---------------------------------------------------------------------------
// RPC URL scheme validation in runDiagnostics
// ---------------------------------------------------------------------------

func TestRPCStatus_InvalidScheme_MarkedUnhealthy(t *testing.T) {
	// Verify that an RPCStatus with a non-http(s) URL and Healthy=false
	// passes ValidateDiagnosticsOutput (i.e. the struct is consistently formed).
	out := &DiagnosticsOutput{
		OverallHealth: "Degraded",
		RPC: []RPCStatus{{
			URL:     "ftp://bad-scheme.example.com",
			Status:  "[FAIL]",
			Healthy: false,
			Error:   "URL must use http:// or https:// scheme — Fix: update rpc_url in config",
		}},
		Cache:  CacheStatus{Healthy: true},
		Config: ConfigSummary{Healthy: true},
	}
	issues := ValidateDiagnosticsOutput(out)
	if len(issues) != 0 {
		t.Errorf("consistently-formed invalid-scheme entry should have no validation issues, got: %v", issues)
	}
}

// ---------------------------------------------------------------------------
// printDiagnosticsDashboard — fix hint in RPC failure
// ---------------------------------------------------------------------------

func TestPrintDiagnosticsDashboard_RPCFailureContainsFixHint(t *testing.T) {
	out := DiagnosticsOutput{
		Version:       "1.0.0",
		Timestamp:     time.Now(),
		OverallHealth: "Degraded",
		System:        SystemInfo{OS: "linux", Arch: "amd64"},
		RPC: []RPCStatus{{
			URL:     "https://bad.example.com",
			Status:  "[FAIL]",
			Healthy: false,
			Error:   "connection refused — Fix: ensure the endpoint is reachable",
		}},
		Cache:  CacheStatus{Healthy: true, Size: "0 B", MaxSize: "500 MB"},
		Config: ConfigSummary{Healthy: true},
	}

	var buf strings.Builder
	// printDiagnosticsDashboard takes io.Writer — wrap the strings.Builder
	printDiagnosticsDashboard(&stringWriterAdapter{&buf}, out)
	rendered := buf.String()

	if !strings.Contains(rendered, "Fix:") {
		t.Errorf("RPC failure section should include a Fix: hint, got:\n%s", rendered)
	}
}

// stringWriterAdapter wraps *strings.Builder to satisfy io.Writer.
type stringWriterAdapter struct{ b *strings.Builder }

func (w *stringWriterAdapter) Write(p []byte) (int, error) { return w.b.Write(p) }
