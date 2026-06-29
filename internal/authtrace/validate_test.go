// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

package authtrace

import (
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// ValidateAuthTraceInputs
// ---------------------------------------------------------------------------

func validHash() string {
	return "5c0a1234567890abcdef1234567890abcdef1234567890abcdef1234567890ab"
}

func TestValidateAuthTraceInputs_Valid(t *testing.T) {
	if err := ValidateAuthTraceInputs(validHash(), "testnet", ""); err != nil {
		t.Errorf("valid inputs should pass, got: %v", err)
	}
}

func TestValidateAuthTraceInputs_EmptyHash(t *testing.T) {
	err := ValidateAuthTraceInputs("", "testnet", "")
	if err == nil {
		t.Fatal("empty hash should fail")
	}
	if !strings.Contains(err.Error(), "transaction hash is required") {
		t.Errorf("error should mention missing hash, got: %v", err)
	}
}

func TestValidateAuthTraceInputs_ShortHash(t *testing.T) {
	err := ValidateAuthTraceInputs("abc123", "testnet", "")
	if err == nil {
		t.Fatal("short hash should fail")
	}
	if !strings.Contains(err.Error(), "exactly 64 hexadecimal") {
		t.Errorf("error should mention 64 chars, got: %v", err)
	}
}

func TestValidateAuthTraceInputs_NonHexHash(t *testing.T) {
	hash := strings.Repeat("z", 64) // z is not hex
	err := ValidateAuthTraceInputs(hash, "", "")
	if err == nil {
		t.Fatal("non-hex hash should fail")
	}
	if !strings.Contains(err.Error(), "non-hexadecimal") {
		t.Errorf("error should mention non-hexadecimal, got: %v", err)
	}
}

func TestValidateAuthTraceInputs_InvalidNetwork(t *testing.T) {
	err := ValidateAuthTraceInputs(validHash(), "devnet", "")
	if err == nil {
		t.Fatal("invalid network should fail")
	}
	if !strings.Contains(err.Error(), "--network") {
		t.Errorf("error should mention --network, got: %v", err)
	}
}

func TestValidateAuthTraceInputs_ValidNetworks(t *testing.T) {
	for _, n := range []string{"testnet", "mainnet", "futurenet", "TESTNET"} {
		if err := ValidateAuthTraceInputs(validHash(), n, ""); err != nil {
			t.Errorf("network %q should be valid, got: %v", n, err)
		}
	}
}

func TestValidateAuthTraceInputs_EmptyNetworkAllowed(t *testing.T) {
	// network auto-detected when empty
	if err := ValidateAuthTraceInputs(validHash(), "", ""); err != nil {
		t.Errorf("empty network should be allowed, got: %v", err)
	}
}

func TestValidateAuthTraceInputs_InvalidRPCURL_NoScheme(t *testing.T) {
	err := ValidateAuthTraceInputs(validHash(), "", "soroban-testnet.stellar.org")
	if err == nil {
		t.Fatal("URL without scheme should fail")
	}
	if !strings.Contains(err.Error(), "http://") {
		t.Errorf("error should mention http scheme, got: %v", err)
	}
}

func TestValidateAuthTraceInputs_InvalidRPCURL_SchemeOnly(t *testing.T) {
	err := ValidateAuthTraceInputs(validHash(), "", "https://")
	if err == nil {
		t.Fatal("URL with scheme but no host should fail")
	}
	if !strings.Contains(err.Error(), "missing host") {
		t.Errorf("error should mention missing host, got: %v", err)
	}
}

func TestValidateAuthTraceInputs_ValidRPCURL(t *testing.T) {
	if err := ValidateAuthTraceInputs(validHash(), "", "https://soroban-testnet.stellar.org"); err != nil {
		t.Errorf("valid RPC URL should pass, got: %v", err)
	}
}

func TestValidateAuthTraceInputs_MultipleErrors(t *testing.T) {
	err := ValidateAuthTraceInputs("", "badnet", "not-a-url")
	if err == nil {
		t.Fatal("multiple invalid inputs should fail")
	}
	if !strings.Contains(err.Error(), "auth validation error") {
		t.Errorf("expected multi-error message, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// ValidateAuthTrace
// ---------------------------------------------------------------------------

func TestValidateAuthTrace_Nil(t *testing.T) {
	err := ValidateAuthTrace(nil)
	if err == nil {
		t.Fatal("nil trace should fail")
	}
	if !strings.Contains(err.Error(), "nil") {
		t.Errorf("error should mention nil, got: %v", err)
	}
}

func TestValidateAuthTrace_NoAccountID(t *testing.T) {
	trace := &AuthTrace{
		AccountID:  "",
		AuthEvents: []AuthEvent{{EventType: "signature_verification", Status: "valid"}},
	}
	err := ValidateAuthTrace(trace)
	if err == nil {
		t.Fatal("missing AccountID should fail")
	}
	if !strings.Contains(err.Error(), "AccountID") {
		t.Errorf("error should mention AccountID, got: %v", err)
	}
}

func TestValidateAuthTrace_NoEvents(t *testing.T) {
	trace := &AuthTrace{AccountID: "GTEST", AuthEvents: nil}
	err := ValidateAuthTrace(trace)
	if err == nil {
		t.Fatal("no events should produce warning")
	}
	if !strings.Contains(err.Error(), "no authorization events") {
		t.Errorf("error should mention no events, got: %v", err)
	}
}

func TestValidateAuthTrace_EventMissingType(t *testing.T) {
	trace := &AuthTrace{
		AccountID: "GTEST",
		AuthEvents: []AuthEvent{
			{EventType: "", Status: "valid"},
		},
	}
	err := ValidateAuthTrace(trace)
	if err == nil {
		t.Fatal("event with no type should fail")
	}
	if !strings.Contains(err.Error(), "no EventType") {
		t.Errorf("error should mention EventType, got: %v", err)
	}
}

func TestValidateAuthTrace_EventMissingStatus(t *testing.T) {
	trace := &AuthTrace{
		AccountID: "GTEST",
		AuthEvents: []AuthEvent{
			{EventType: "signature_verification", Status: ""},
		},
	}
	err := ValidateAuthTrace(trace)
	if err == nil {
		t.Fatal("event with no status should fail")
	}
	if !strings.Contains(err.Error(), "no Status") {
		t.Errorf("error should mention Status, got: %v", err)
	}
}

func TestValidateAuthTrace_UnknownFailureReason(t *testing.T) {
	trace := &AuthTrace{
		AccountID: "GTEST",
		AuthEvents: []AuthEvent{
			{EventType: "threshold_check", Status: "failed"},
		},
		Failures: []AuthFailure{
			{AccountID: "GTEST", FailureReason: ReasonUnknown, RequiredWeight: 2, CollectedWeight: 1},
		},
	}
	err := ValidateAuthTrace(trace)
	if err == nil {
		t.Fatal("unknown failure reason should fail")
	}
	if !strings.Contains(err.Error(), "unknown failure reason") {
		t.Errorf("error should mention unknown failure reason, got: %v", err)
	}
}

func TestValidateAuthTrace_InconsistentWeights(t *testing.T) {
	trace := &AuthTrace{
		AccountID: "GTEST",
		AuthEvents: []AuthEvent{
			{EventType: "threshold_check", Status: "failed"},
		},
		Failures: []AuthFailure{
			{AccountID: "GTEST", FailureReason: ReasonThresholdNotMet,
				RequiredWeight: 2, CollectedWeight: 5}, // collected > required but still failed
		},
	}
	err := ValidateAuthTrace(trace)
	if err == nil {
		t.Fatal("inconsistent weights should fail")
	}
	if !strings.Contains(err.Error(), "inconsistent") {
		t.Errorf("error should mention inconsistent, got: %v", err)
	}
}

func TestValidateAuthTrace_Valid(t *testing.T) {
	trace := &AuthTrace{
		AccountID: "GTEST",
		AuthEvents: []AuthEvent{
			{EventType: "signature_verification", Status: "valid"},
		},
	}
	if err := ValidateAuthTrace(trace); err != nil {
		t.Errorf("valid trace should pass, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// ValidateSignatureInput
// ---------------------------------------------------------------------------

func TestValidateSignatureInput_Valid(t *testing.T) {
	if err := ValidateSignatureInput("GTEST", "key1", Ed25519); err != nil {
		t.Errorf("valid input should pass, got: %v", err)
	}
}

func TestValidateSignatureInput_EmptyAccountID(t *testing.T) {
	err := ValidateSignatureInput("", "key1", Ed25519)
	if err == nil {
		t.Fatal("empty accountID should fail")
	}
	if !strings.Contains(err.Error(), "accountID") {
		t.Errorf("error should mention accountID, got: %v", err)
	}
}

func TestValidateSignatureInput_EmptySignerKey(t *testing.T) {
	err := ValidateSignatureInput("GTEST", "", Ed25519)
	if err == nil {
		t.Fatal("empty signerKey should fail")
	}
	if !strings.Contains(err.Error(), "signerKey") {
		t.Errorf("error should mention signerKey, got: %v", err)
	}
}

func TestValidateSignatureInput_UnknownSigType(t *testing.T) {
	err := ValidateSignatureInput("GTEST", "key1", SignatureType("quantum"))
	if err == nil {
		t.Fatal("unknown sig type should fail")
	}
	if !strings.Contains(err.Error(), "unknown signature type") {
		t.Errorf("error should mention unknown signature type, got: %v", err)
	}
}

func TestValidateSignatureInput_AllKnownTypes(t *testing.T) {
	for _, st := range []SignatureType{Ed25519, Secp256k1, PreAuthorized, CustomAccount} {
		if err := ValidateSignatureInput("GTEST", "key1", st); err != nil {
			t.Errorf("type %q should be valid, got: %v", st, err)
		}
	}
}

// ---------------------------------------------------------------------------
// ValidateContractAuthInput
// ---------------------------------------------------------------------------

func TestValidateContractAuthInput_Valid(t *testing.T) {
	if err := ValidateContractAuthInput("CABC123", "transfer"); err != nil {
		t.Errorf("valid inputs should pass, got: %v", err)
	}
}

func TestValidateContractAuthInput_EmptyContractID(t *testing.T) {
	err := ValidateContractAuthInput("", "transfer")
	if err == nil {
		t.Fatal("empty contractID should fail")
	}
	if !strings.Contains(err.Error(), "contractID") {
		t.Errorf("error should mention contractID, got: %v", err)
	}
}

func TestValidateContractAuthInput_EmptyMethod(t *testing.T) {
	err := ValidateContractAuthInput("CABC123", "")
	if err == nil {
		t.Fatal("empty method should fail")
	}
	if !strings.Contains(err.Error(), "method") {
		t.Errorf("error should mention method, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// GenerateTrace — AccountID and SignerCount populated
// ---------------------------------------------------------------------------

func TestGenerateTrace_PopulatesAccountID(t *testing.T) {
	tracker := NewTracker(Config{})
	tracker.InitializeAccountContext("GTEST",
		[]SignerInfo{{AccountID: "GTEST", SignerKey: "k1", SignerType: Ed25519, Weight: 1}},
		ThresholdConfig{HighThreshold: 1},
	)
	trace := tracker.GenerateTrace()
	if trace.AccountID == "" {
		t.Error("GenerateTrace should populate AccountID from context")
	}
	if trace.SignerCount != 1 {
		t.Errorf("expected SignerCount=1, got %d", trace.SignerCount)
	}
}

// ---------------------------------------------------------------------------
// DetailedReporter — nil trace guard
// ---------------------------------------------------------------------------

func TestDetailedReporter_NilTrace_GenerateReport(t *testing.T) {
	r := &DetailedReporter{trace: nil}
	report := r.GenerateReport()
	if !strings.Contains(report, "Error") {
		t.Errorf("nil trace should return error message, got: %q", report)
	}
}

func TestDetailedReporter_NilTrace_GenerateJSON(t *testing.T) {
	r := &DetailedReporter{trace: nil}
	_, err := r.GenerateJSON()
	if err == nil {
		t.Fatal("nil trace should return error from GenerateJSON")
	}
	if !strings.Contains(err.Error(), "nil") {
		t.Errorf("error should mention nil, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// ValidateContract — input validation wired in
// ---------------------------------------------------------------------------

func TestValidateContract_EmptyContractID(t *testing.T) {
	v := NewCustomContractAuthValidator()
	_, err := v.ValidateContract("", "transfer", nil)
	if err == nil {
		t.Fatal("empty contractID should fail")
	}
	if !strings.Contains(err.Error(), "contractID") {
		t.Errorf("error should mention contractID, got: %v", err)
	}
}

func TestValidateContract_EmptyMethod(t *testing.T) {
	v := NewCustomContractAuthValidator()
	_, err := v.ValidateContract("CABC", "", nil)
	if err == nil {
		t.Fatal("empty method should fail")
	}
	if !strings.Contains(err.Error(), "method") {
		t.Errorf("error should mention method, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// AuthInputError formatting
// ---------------------------------------------------------------------------

func TestAuthInputError_Single(t *testing.T) {
	e := &AuthInputError{Failures: []string{"something failed"}}
	if e.Error() != "something failed" {
		t.Errorf("single failure should return bare message, got: %q", e.Error())
	}
}

func TestAuthInputError_Multiple(t *testing.T) {
	e := &AuthInputError{Failures: []string{"err one", "err two"}}
	msg := e.Error()
	if !strings.Contains(msg, "2 auth validation error") {
		t.Errorf("expected error count in message, got: %q", msg)
	}
	if !strings.Contains(msg, "1. err one") {
		t.Errorf("expected enumeration, got: %q", msg)
	}
}
