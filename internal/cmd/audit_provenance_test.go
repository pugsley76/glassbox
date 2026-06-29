// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"strings"
	"testing"
	"time"
)

// generateSelfSignedCertPEM creates a minimal self-signed certificate PEM for tests.
func generateSelfSignedCertPEM(t *testing.T) string {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "test"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, pub, priv)
	if err != nil {
		t.Fatal(err)
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))
}

func TestBuildProvenance_NilWhenNoFlags(t *testing.T) {
	// Reset all provenance flags.
	auditSignSignerIdentity = ""
	auditSignKeyID = ""
	auditSignCertChainFile = ""
	auditSignPreviousSignatureHash = ""

	prov, err := buildProvenance("software", "", "ed25519")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if prov != nil {
		t.Fatalf("expected nil provenance when no flags set, got %+v", prov)
	}
}

func TestBuildProvenance_FieldsPopulated(t *testing.T) {
	auditSignSignerIdentity = "ops@example.com"
	auditSignKeyID = "my-key-label"
	auditSignCertChainFile = ""
	auditSignPreviousSignatureHash = strings.Repeat("a", 64)
	defer func() {
		auditSignSignerIdentity = ""
		auditSignKeyID = ""
		auditSignPreviousSignatureHash = ""
	}()

	prov, err := buildProvenance("software", "", "ed25519")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if prov == nil {
		t.Fatal("expected non-nil provenance")
	}
	if prov.SignerIdentity != "ops@example.com" {
		t.Errorf("SignerIdentity = %q, want %q", prov.SignerIdentity, "ops@example.com")
	}
	if prov.KeyID != "my-key-label" {
		t.Errorf("KeyID = %q, want %q", prov.KeyID, "my-key-label")
	}
	// Algorithm is now passed in directly rather than inferred from provider name.
	if prov.Algorithm != "ed25519" {
		t.Errorf("Algorithm = %q, want ed25519", prov.Algorithm)
	}
	if prov.PreviousSignatureHash != strings.Repeat("a", 64) {
		t.Errorf("PreviousSignatureHash mismatch")
	}
}

func TestBuildProvenance_AlgorithmPassedThrough(t *testing.T) {
	// Algorithm is now derived from the live signer, not the provider name.
	// Verify that whatever algorithm string is passed in is preserved.
	auditSignSignerIdentity = "hsm-user"
	defer func() { auditSignSignerIdentity = "" }()

	prov, err := buildProvenance("pkcs11", "", "ed25519")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if prov.Algorithm != "ed25519" {
		t.Errorf("Algorithm = %q, want ed25519 (from signer, not inferred from provider name)", prov.Algorithm)
	}
}

func TestValidateProvenance_NilIsValid(t *testing.T) {
	ok, err := validateProvenance(nil)
	if !ok || err != nil {
		t.Errorf("nil provenance should be valid, got ok=%v err=%v", ok, err)
	}
}

func TestValidateProvenance_ValidCertChain(t *testing.T) {
	certPEM := generateSelfSignedCertPEM(t)
	prov := &SignatureProvenance{
		CertificateChain: []string{certPEM},
	}
	ok, err := validateProvenance(prov)
	if !ok || err != nil {
		t.Errorf("valid cert chain should pass, got ok=%v err=%v", ok, err)
	}
}

func TestValidateProvenance_InvalidCertChain(t *testing.T) {
	prov := &SignatureProvenance{
		CertificateChain: []string{"not-valid-pem"},
	}
	ok, err := validateProvenance(prov)
	if ok || err == nil {
		t.Error("invalid PEM should fail provenance validation")
	}
}

func TestValidateProvenance_ValidPreviousHash(t *testing.T) {
	hash := hex.EncodeToString(make([]byte, 32))
	prov := &SignatureProvenance{PreviousSignatureHash: hash}
	ok, err := validateProvenance(prov)
	if !ok || err != nil {
		t.Errorf("valid hash should pass, got ok=%v err=%v", ok, err)
	}
}

func TestValidateProvenance_InvalidPreviousHash(t *testing.T) {
	prov := &SignatureProvenance{PreviousSignatureHash: "tooshort"}
	ok, err := validateProvenance(prov)
	if ok || err == nil {
		t.Error("short hash should fail provenance validation")
	}
}

func TestSignedAuditLog_ProvenanceRoundTrip(t *testing.T) {
	certPEM := generateSelfSignedCertPEM(t)
	log := SignedAuditLog{
		Version:   "1.0.0",
		Timestamp: time.Now().UTC(),
		TraceHash: strings.Repeat("b", 64),
		Signature: strings.Repeat("c", 128),
		PublicKey: strings.Repeat("d", 64),
		Provider:  "software",
		Provenance: &SignatureProvenance{
			SignerIdentity:        "ci@example.com",
			KeyID:                 "key-001",
			Algorithm:             "Ed25519",
			CertificateChain:      []string{certPEM},
			PreviousSignatureHash: strings.Repeat("e", 64),
		},
		Payload: json.RawMessage(`{}`),
	}

	data, err := json.Marshal(log)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var decoded SignedAuditLog
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if decoded.Provenance == nil {
		t.Fatal("provenance lost after round-trip")
	}
	if decoded.Provenance.SignerIdentity != "ci@example.com" {
		t.Errorf("SignerIdentity mismatch after round-trip")
	}
	if len(decoded.Provenance.CertificateChain) != 1 {
		t.Errorf("CertificateChain length mismatch after round-trip")
	}
}

func TestAuditSign_ProvenanceInOutput(t *testing.T) {
	resetAuditSignFlags()
	auditSignPayload = `{"input":{},"state":{},"events":[],"timestamp":"2026-01-01T00:00:00.000Z"}`
	auditSignSoftwareKey = generateTestPEM(t)
	auditSignSignerIdentity = "test-signer"
	auditSignKeyID = "test-key-id"
	defer resetAuditSignFlags()

	var buf bytes.Buffer
	auditSignCmd.SetOut(&buf)
	err := auditSignCmd.RunE(auditSignCmd, nil)
	if err != nil {
		t.Fatalf("audit:sign failed: %v", err)
	}

	var result SignedAuditLog
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse output: %v", err)
	}
	if result.Provenance == nil {
		t.Fatal("expected provenance in output")
	}
	if result.Provenance.SignerIdentity != "test-signer" {
		t.Errorf("SignerIdentity = %q, want test-signer", result.Provenance.SignerIdentity)
	}
	if result.Provenance.Algorithm != "Ed25519" {
		t.Errorf("Algorithm = %q, want Ed25519", result.Provenance.Algorithm)
	}
}
