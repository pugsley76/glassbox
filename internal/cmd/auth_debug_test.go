// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"strings"
	"testing"

	"github.com/dotandev/glassbox/internal/authtrace"
)

// validAuthTxHash is a well-formed 64-character hex transaction hash used across
// the auth-debug validation tests.
const validAuthTxHash = "5c0a1234567890abcdef1234567890abcdef1234567890abcdef1234567890ab"

// TestValidateAuthDebugInputs_TransactionHash verifies that the transaction hash
// is validated before any network access, rejecting malformed values with an
// explicit message and accepting well-formed ones.
func TestValidateAuthDebugInputs_TransactionHash(t *testing.T) {
	tests := []struct {
		name       string
		txHash     string
		wantErr    bool
		wantSubstr string
	}{
		{
			name:    "valid hash",
			txHash:  validAuthTxHash,
			wantErr: false,
		},
		{
			name:       "too short",
			txHash:     "abc123",
			wantErr:    true,
			wantSubstr: "transaction hash",
		},
		{
			name:       "empty",
			txHash:     "",
			wantErr:    true,
			wantSubstr: "transaction hash",
		},
		{
			name:       "correct length but non-hex",
			txHash:     strings.Repeat("z", 64),
			wantErr:    true,
			wantSubstr: "hexadecimal",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			err := validateAuthDebugInputs(tt.txHash, "testnet", "")
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error for hash %q, got nil", tt.txHash)
				}
				if tt.wantSubstr != "" && !strings.Contains(err.Error(), tt.wantSubstr) {
					t.Errorf("error %q does not contain %q", err.Error(), tt.wantSubstr)
				}
				// The offending value should be echoed back to the user.
				if tt.txHash != "" && !strings.Contains(err.Error(), tt.txHash) {
					t.Errorf("error %q does not echo the invalid hash %q", err.Error(), tt.txHash)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for valid hash: %v", err)
			}
		})
	}
}

// TestValidateAuthDebugInputs_RPCURL verifies that a malformed --rpc-url is
// rejected early, while an empty value (use defaults) and valid URLs pass.
func TestValidateAuthDebugInputs_RPCURL(t *testing.T) {
	tests := []struct {
		name    string
		rpcURL  string
		wantErr bool
	}{
		{name: "empty is allowed", rpcURL: "", wantErr: false},
		{name: "valid https", rpcURL: "https://horizon-testnet.stellar.org", wantErr: false},
		{name: "valid comma-separated fallback", rpcURL: "https://a.example.org,https://b.example.org", wantErr: false},
		{name: "unsupported scheme", rpcURL: "ftp://horizon.example.org", wantErr: true},
		{name: "missing host", rpcURL: "https://", wantErr: true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			err := validateAuthDebugInputs(validAuthTxHash, "testnet", tt.rpcURL)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error for --rpc-url %q, got nil", tt.rpcURL)
				}
				if !strings.Contains(err.Error(), "rpc-url") {
					t.Errorf("error %q does not mention --rpc-url", err.Error())
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for --rpc-url %q: %v", tt.rpcURL, err)
			}
		})
	}
}

// TestValidateAuthDebugInputs_Network verifies built-in networks are accepted
// and an unknown network is rejected with an actionable message.
func TestValidateAuthDebugInputs_Network(t *testing.T) {
	for _, net := range []string{"testnet", "mainnet", "futurenet"} {
		net := net
		t.Run("valid_"+net, func(t *testing.T) {
			if err := validateAuthDebugInputs(validAuthTxHash, net, ""); err != nil {
				t.Errorf("network %q should be accepted, got: %v", net, err)
			}
		})
	}

	t.Run("invalid network rejected", func(t *testing.T) {
		err := validateAuthDebugInputs(validAuthTxHash, "badnet", "")
		if err == nil {
			t.Fatal("expected error for unknown network")
		}
		if !strings.Contains(err.Error(), "badnet") {
			t.Errorf("error %q does not echo the invalid network", err.Error())
		}
	})
}

// TestAuthTraceHasData verifies the empty-trace detection used to attach the
// "no authorization events" diagnostic.
func TestAuthTraceHasData(t *testing.T) {
	tests := []struct {
		name  string
		trace *authtrace.AuthTrace
		want  bool
	}{
		{name: "nil trace", trace: nil, want: false},
		{name: "empty trace", trace: &authtrace.AuthTrace{}, want: false},
		{
			name:  "has events",
			trace: &authtrace.AuthTrace{AuthEvents: []authtrace.AuthEvent{{EventType: "signature_verification"}}},
			want:  true,
		},
		{
			name:  "has failures",
			trace: &authtrace.AuthTrace{Failures: []authtrace.AuthFailure{{AccountID: "GABC"}}},
			want:  true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if got := authTraceHasData(tt.trace); got != tt.want {
				t.Errorf("authTraceHasData = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestEmptyAuthTraceNote verifies the diagnostic echoes the transaction hash and
// makes clear that the report is not a verified-successful authorization.
func TestEmptyAuthTraceNote(t *testing.T) {
	note := emptyAuthTraceNote(validAuthTxHash)
	if !strings.Contains(note, validAuthTxHash) {
		t.Errorf("note does not contain the transaction hash: %q", note)
	}
	for _, want := range []string{"no authorization events", "not a verified"} {
		if !strings.Contains(note, want) {
			t.Errorf("note %q is missing expected phrase %q", note, want)
		}
	}
}

// TestAuthDebugPreRunE_RejectsInvalidHash exercises the command's PreRunE end to
// end. The --network flag is set explicitly so the network auto-detection probe
// (which would otherwise reach the network) is skipped.
func TestAuthDebugPreRunE_RejectsInvalidHash(t *testing.T) {
	origNetwork := authNetworkFlag
	origRPCURL := authRPCURLFlag
	t.Cleanup(func() {
		authNetworkFlag = origNetwork
		authRPCURLFlag = origRPCURL
	})

	// Set the flag so cmd.Flags().Changed("network") is true and the probe is skipped.
	if err := authDebugCmd.Flags().Set("network", "testnet"); err != nil {
		t.Fatalf("failed to set network flag: %v", err)
	}

	err := authDebugCmd.PreRunE(authDebugCmd, []string{"not-a-valid-hash"})
	if err == nil {
		t.Fatal("expected PreRunE to reject an invalid transaction hash")
	}
	if !strings.Contains(err.Error(), "transaction hash") {
		t.Errorf("error %q does not mention the transaction hash", err.Error())
	}
}

// TestAuthDebugPreRunE_AcceptsValidInputs verifies that a well-formed hash with
// an explicit valid network passes PreRunE without a validation error.
func TestAuthDebugPreRunE_AcceptsValidInputs(t *testing.T) {
	origNetwork := authNetworkFlag
	origRPCURL := authRPCURLFlag
	t.Cleanup(func() {
		authNetworkFlag = origNetwork
		authRPCURLFlag = origRPCURL
	})

	if err := authDebugCmd.Flags().Set("network", "testnet"); err != nil {
		t.Fatalf("failed to set network flag: %v", err)
	}
	authRPCURLFlag = ""

	if err := authDebugCmd.PreRunE(auditVerifyCmd, []string{validAuthTxHash}); err != nil {
		t.Fatalf("expected valid inputs to pass PreRunE, got: %v", err)
	}
}

// ── Auth trace diagnostics (source mapping) ──────────────────────────────────

func TestAuthTraceDiagnostics_EmptyTrace_ReasonSet(t *testing.T) {
	tracker := authtrace.NewTracker(authtrace.Config{})
	trace := tracker.GenerateTrace()

	if trace.Diagnostics == nil {
		t.Fatal("expected Diagnostics to be set for empty trace")
	}
	if trace.Diagnostics.EmptyTraceReason == "" {
		t.Error("expected EmptyTraceReason to be populated for empty trace")
	}
	if !strings.Contains(trace.Diagnostics.EmptyTraceReason, "no Soroban authorization entries") {
		t.Errorf("EmptyTraceReason should mention Soroban auth entries, got: %s", trace.Diagnostics.EmptyTraceReason)
	}
	if !strings.Contains(trace.Diagnostics.EmptyTraceReason, "doctor") {
		t.Errorf("EmptyTraceReason should suggest running 'glassbox doctor', got: %s", trace.Diagnostics.EmptyTraceReason)
	}
}

func TestAuthTraceDiagnostics_WithEvents_SourceMappingHintPresent(t *testing.T) {
	tracker := authtrace.NewTracker(authtrace.Config{
		TraceCustomContracts: true,
		CaptureSigDetails:    true,
		MaxEventDepth:        1000,
	})
	tracker.RecordEvent(authtrace.AuthEvent{
		EventType: "signature_verification",
		Status:    "valid",
		Weight:    1,
		AccountID: "GABC",
	})
	trace := tracker.GenerateTrace()

	if trace.Diagnostics == nil {
		t.Fatal("expected Diagnostics to be set when events are present")
	}
	if trace.Diagnostics.SourceMappingAvailable {
		t.Error("expected SourceMappingAvailable to be false when events lack source mapping")
	}
	if !strings.Contains(trace.Diagnostics.SourceMappingHint, "debug = true") {
		t.Errorf("SourceMappingHint should suggest recompiling with debug=true, got: %s", trace.Diagnostics.SourceMappingHint)
	}
	if !strings.Contains(trace.Diagnostics.SourceMappingHint, "--contract-source") {
		t.Errorf("SourceMappingHint should suggest --contract-source, got: %s", trace.Diagnostics.SourceMappingHint)
	}
}

func TestAuthTraceDiagnostics_WithSourceMapping_CoverageTracked(t *testing.T) {
	tracker := authtrace.NewTracker(authtrace.Config{})
	tracker.RecordEvent(authtrace.AuthEvent{
		EventType:  "signature_verification",
		Status:     "valid",
		SourceFile: "src/contract.rs",
		SourceLine: 42,
	})
	tracker.RecordEvent(authtrace.AuthEvent{
		EventType: "threshold_check",
		Status:    "fail",
	})
	trace := tracker.GenerateTrace()

	if trace.Diagnostics == nil {
		t.Fatal("expected Diagnostics to be set")
	}
	if !trace.Diagnostics.SourceMappingAvailable {
		t.Error("expected SourceMappingAvailable to be true when at least one event has source context")
	}
	if trace.Diagnostics.EventsWithSourceCount != 1 {
		t.Errorf("expected 1 event with source mapping, got %d", trace.Diagnostics.EventsWithSourceCount)
	}
	if trace.Diagnostics.SourceMappingHint != "" {
		t.Errorf("expected no source mapping hint when source context is available, got: %s", trace.Diagnostics.SourceMappingHint)
	}
}
	authRPCURLFlag = ""

	if err := authDebugCmd.PreRunE(authDebugCmd, []string{validAuthTxHash}); err != nil {
		t.Fatalf("expected valid inputs to pass PreRunE, got: %v", err)
	}
}
