// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

package session

import (
	"strings"
	"testing"
	"time"
)

// validAuthSession returns a fully-valid AuthSession that individual tests
// mutate to exercise a single failure path at a time.
func validAuthSession() *AuthSession {
	start := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	return &AuthSession{
		SessionID:             "sess-abc123",
		TxHash:                strings.Repeat("a", 64),
		Network:               "testnet",
		RPCURL:                "https://horizon-testnet.stellar.org",
		Detailed:              true,
		AuthEventCount:        3,
		FailureCount:          0,
		MissingSignatureCount: 0,
		Status:                "completed",
		StartedAt:             start,
		CompletedAt:           start.Add(2 * time.Second),
	}
}

// TestValidateAuthSession_Valid confirms a well-formed record passes with no
// issues and produces a nil Error().
func TestValidateAuthSession_Valid(t *testing.T) {
	report := ValidateAuthSession(validAuthSession())
	if !report.OK {
		t.Fatalf("expected valid auth session, got issues: %v", report.Issues)
	}
	if err := report.Error(); err != nil {
		t.Errorf("expected nil error for valid report, got: %v", err)
	}
}

// TestValidateAuthSession_NoAuthDataIsValid confirms the "no_auth_data" terminal
// state is accepted when there are no failures and no error.
func TestValidateAuthSession_NoAuthDataIsValid(t *testing.T) {
	as := validAuthSession()
	as.Status = "no_auth_data"
	as.AuthEventCount = 0
	report := ValidateAuthSession(as)
	if !report.OK {
		t.Fatalf("expected no_auth_data session to be valid, got: %v", report.Issues)
	}
}

// TestValidateAuthSession_Nil confirms a nil record is reported rather than
// panicking.
func TestValidateAuthSession_Nil(t *testing.T) {
	report := ValidateAuthSession(nil)
	if report.OK {
		t.Fatal("expected nil auth session to be invalid")
	}
	if report.Error() == nil {
		t.Error("expected an error for a nil auth session")
	}
}

// TestValidateAuthSession_Failures table-tests each individual validation rule.
func TestValidateAuthSession_Failures(t *testing.T) {
	tests := []struct {
		name       string
		mutate     func(*AuthSession)
		wantField  string
		wantSubstr string
	}{
		{
			name:       "missing session id",
			mutate:     func(a *AuthSession) { a.SessionID = "" },
			wantField:  "SessionID",
			wantSubstr: "parent session ID",
		},
		{
			name:       "empty tx hash",
			mutate:     func(a *AuthSession) { a.TxHash = "" },
			wantField:  "TxHash",
			wantSubstr: "missing the transaction hash",
		},
		{
			name:       "tx hash wrong length",
			mutate:     func(a *AuthSession) { a.TxHash = "abc123" },
			wantField:  "TxHash",
			wantSubstr: "exactly 64 characters",
		},
		{
			name:       "tx hash non-hex",
			mutate:     func(a *AuthSession) { a.TxHash = strings.Repeat("z", 64) },
			wantField:  "TxHash",
			wantSubstr: "non-hexadecimal",
		},
		{
			name:       "missing network",
			mutate:     func(a *AuthSession) { a.Network = "" },
			wantField:  "Network",
			wantSubstr: "missing the network",
		},
		{
			name:       "unknown network",
			mutate:     func(a *AuthSession) { a.Network = "devnet" },
			wantField:  "Network",
			wantSubstr: "not a recognised Stellar network",
		},
		{
			name:       "bad rpc scheme",
			mutate:     func(a *AuthSession) { a.RPCURL = "ftp://example.org" },
			wantField:  "RPCURL",
			wantSubstr: "scheme must be http or https",
		},
		{
			name:       "rpc missing host",
			mutate:     func(a *AuthSession) { a.RPCURL = "https://" },
			wantField:  "RPCURL",
			wantSubstr: "missing host",
		},
		{
			name:       "zero started_at",
			mutate:     func(a *AuthSession) { a.StartedAt = time.Time{} },
			wantField:  "StartedAt",
			wantSubstr: "zero started_at",
		},
		{
			name: "completed before started",
			mutate: func(a *AuthSession) {
				a.CompletedAt = a.StartedAt.Add(-time.Hour)
			},
			wantField:  "CompletedAt",
			wantSubstr: "before started_at",
		},
		{
			name:       "negative auth event count",
			mutate:     func(a *AuthSession) { a.AuthEventCount = -1 },
			wantField:  "AuthEventCount",
			wantSubstr: "is negative",
		},
		{
			name:       "negative failure count",
			mutate:     func(a *AuthSession) { a.FailureCount = -2 },
			wantField:  "FailureCount",
			wantSubstr: "is negative",
		},
		{
			name:       "missing status",
			mutate:     func(a *AuthSession) { a.Status = "" },
			wantField:  "Status",
			wantSubstr: "missing a status",
		},
		{
			name:       "unknown status",
			mutate:     func(a *AuthSession) { a.Status = "bogus" },
			wantField:  "Status",
			wantSubstr: "not recognised",
		},
		{
			name:       "failed without error",
			mutate:     func(a *AuthSession) { a.Status = "failed"; a.Error = "" },
			wantField:  "Error",
			wantSubstr: "no error message",
		},
		{
			name: "non-failed with error set",
			mutate: func(a *AuthSession) {
				a.Status = "completed"
				a.Error = "boom"
			},
			wantField:  "Error",
			wantSubstr: "an error message is set",
		},
		{
			name: "no_auth_data with failures",
			mutate: func(a *AuthSession) {
				a.Status = "no_auth_data"
				a.FailureCount = 2
			},
			wantField:  "FailureCount",
			wantSubstr: "no_auth_data",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			as := validAuthSession()
			tt.mutate(as)
			report := ValidateAuthSession(as)
			if report.OK {
				t.Fatalf("expected validation failure for %q, but report was OK", tt.name)
			}
			if !hasIssueField(report, tt.wantField) {
				t.Errorf("expected an issue on field %q, got issues: %v", tt.wantField, report.Issues)
			}
			if err := report.Error(); err == nil || !strings.Contains(err.Error(), tt.wantSubstr) {
				t.Errorf("error %v does not contain %q", err, tt.wantSubstr)
			}
		})
	}
}

// TestValidateAuthSession_FailedWithErrorIsValid confirms a failed run that
// records its error is considered consistent.
func TestValidateAuthSession_FailedWithErrorIsValid(t *testing.T) {
	as := validAuthSession()
	as.Status = "failed"
	as.Error = "transaction not found"
	as.CompletedAt = time.Time{}
	report := ValidateAuthSession(as)
	if !report.OK {
		t.Fatalf("expected a failed-with-error session to be valid, got: %v", report.Issues)
	}
}

// TestValidateAuthSession_AggregatesMultipleIssues confirms several independent
// problems are all surfaced in a single report rather than only the first.
func TestValidateAuthSession_AggregatesMultipleIssues(t *testing.T) {
	as := &AuthSession{} // empty: missing session id, tx hash, network, started_at, status
	report := ValidateAuthSession(as)
	if report.OK {
		t.Fatal("expected an empty auth session to be invalid")
	}
	if len(report.Issues) < 4 {
		t.Errorf("expected multiple issues for an empty session, got %d: %v", len(report.Issues), report.Issues)
	}
	err := report.Error()
	if err == nil || !strings.Contains(err.Error(), "problems") {
		t.Errorf("aggregated error should mention the problem count, got: %v", err)
	}
}

// hasIssueField reports whether the report contains an issue on the given field.
func hasIssueField(report *AuthSessionReport, field string) bool {
	for _, issue := range report.Issues {
		if issue.Field == field {
			return true
		}
	}
	return false
}
