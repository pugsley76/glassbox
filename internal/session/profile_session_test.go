// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

package session

import (
	"strings"
	"testing"
	"time"
)

func validProfileSession() *ProfileSession {
	now := time.Now()
	return &ProfileSession{
		SessionID:   "session-abc123",
		TxHash:      "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		Network:     "testnet",
		Format:      "html",
		OutputPath:  "flamegraph.html",
		StartedAt:   now.Add(-30 * time.Second),
		CompletedAt: now,
	}
}

func TestValidateProfileSession_Valid_OK(t *testing.T) {
	report := ValidateProfileSession(validProfileSession())
	if !report.OK {
		t.Errorf("expected OK=true for valid profile session, got issues: %v", report.Issues)
	}
}

func TestValidateProfileSession_MissingSessionID_ReportsIssue(t *testing.T) {
	ps := validProfileSession()
	ps.SessionID = ""
	report := ValidateProfileSession(ps)
	if report.OK {
		t.Fatal("expected OK=false when SessionID is empty")
	}
	requireProfileIssueField(t, report, "SessionID")
}

func TestValidateProfileSession_MissingTxHash_ReportsIssue(t *testing.T) {
	ps := validProfileSession()
	ps.TxHash = ""
	report := ValidateProfileSession(ps)
	if report.OK {
		t.Fatal("expected OK=false when TxHash is empty")
	}
	requireProfileIssueField(t, report, "TxHash")
}

func TestValidateProfileSession_MissingNetwork_ReportsIssue(t *testing.T) {
	ps := validProfileSession()
	ps.Network = ""
	report := ValidateProfileSession(ps)
	if report.OK {
		t.Fatal("expected OK=false when Network is empty")
	}
	requireProfileIssueField(t, report, "Network")
}

func TestValidateProfileSession_InvalidNetwork_ReportsIssue(t *testing.T) {
	ps := validProfileSession()
	ps.Network = "devnet"
	report := ValidateProfileSession(ps)
	if report.OK {
		t.Fatal("expected OK=false for unrecognised network")
	}
	requireProfileIssueField(t, report, "Network")
	for _, issue := range report.Issues {
		if issue.Field == "Network" && !strings.Contains(issue.Description, "devnet") {
			t.Errorf("issue description should name the bad value 'devnet', got: %q", issue.Description)
		}
	}
}

func TestValidateProfileSession_ValidNetworks_OK(t *testing.T) {
	for _, net := range []string{"testnet", "mainnet", "futurenet"} {
		ps := validProfileSession()
		ps.Network = net
		report := ValidateProfileSession(ps)
		for _, issue := range report.Issues {
			if issue.Field == "Network" {
				t.Errorf("valid network %q should not produce a Network issue, got: %v", net, issue)
			}
		}
	}
}

func TestValidateProfileSession_MissingFormat_ReportsIssue(t *testing.T) {
	ps := validProfileSession()
	ps.Format = ""
	report := ValidateProfileSession(ps)
	if report.OK {
		t.Fatal("expected OK=false when Format is empty")
	}
	requireProfileIssueField(t, report, "Format")
}

func TestValidateProfileSession_InvalidFormat_ReportsIssue(t *testing.T) {
	ps := validProfileSession()
	ps.Format = "pdf"
	report := ValidateProfileSession(ps)
	if report.OK {
		t.Fatal("expected OK=false for unsupported format")
	}
	requireProfileIssueField(t, report, "Format")
	for _, issue := range report.Issues {
		if issue.Field == "Format" && !strings.Contains(issue.Description, "pdf") {
			t.Errorf("issue description should name the bad format, got: %q", issue.Description)
		}
	}
}

func TestValidateProfileSession_ValidFormats_OK(t *testing.T) {
	for _, fmt := range SupportedProfileFormats {
		ps := validProfileSession()
		ps.Format = fmt
		report := ValidateProfileSession(ps)
		for _, issue := range report.Issues {
			if issue.Field == "Format" {
				t.Errorf("valid format %q should not produce a Format issue, got: %v", fmt, issue)
			}
		}
	}
}

func TestValidateProfileSession_ZeroStartedAt_ReportsIssue(t *testing.T) {
	ps := validProfileSession()
	ps.StartedAt = time.Time{}
	report := ValidateProfileSession(ps)
	if report.OK {
		t.Fatal("expected OK=false when StartedAt is zero")
	}
	requireProfileIssueField(t, report, "StartedAt")
}

func TestValidateProfileSession_CompletedBeforeStarted_ReportsIssue(t *testing.T) {
	ps := validProfileSession()
	ps.StartedAt = time.Now()
	ps.CompletedAt = time.Now().Add(-time.Minute)
	report := ValidateProfileSession(ps)
	if report.OK {
		t.Fatal("expected OK=false when CompletedAt is before StartedAt")
	}
	requireProfileIssueField(t, report, "CompletedAt")
}

func TestValidateProfileSession_ZeroCompletedAt_OK(t *testing.T) {
	// CompletedAt being zero is valid (in-progress profile).
	ps := validProfileSession()
	ps.CompletedAt = time.Time{}
	report := ValidateProfileSession(ps)
	for _, issue := range report.Issues {
		if issue.Field == "CompletedAt" {
			t.Errorf("zero CompletedAt should not produce an issue, got: %v", issue)
		}
	}
}

func TestValidateProfileSession_AllIssuesHaveHints(t *testing.T) {
	scenarios := []*ProfileSession{
		func() *ProfileSession { ps := validProfileSession(); ps.SessionID = ""; return ps }(),
		func() *ProfileSession { ps := validProfileSession(); ps.TxHash = ""; return ps }(),
		func() *ProfileSession { ps := validProfileSession(); ps.Network = ""; return ps }(),
		func() *ProfileSession { ps := validProfileSession(); ps.Network = "badnet"; return ps }(),
		func() *ProfileSession { ps := validProfileSession(); ps.Format = ""; return ps }(),
		func() *ProfileSession { ps := validProfileSession(); ps.Format = "pdf"; return ps }(),
		func() *ProfileSession { ps := validProfileSession(); ps.StartedAt = time.Time{}; return ps }(),
		func() *ProfileSession {
			ps := validProfileSession()
			ps.StartedAt = time.Now()
			ps.CompletedAt = time.Now().Add(-time.Minute)
			return ps
		}(),
	}
	for i, ps := range scenarios {
		report := ValidateProfileSession(ps)
		for _, issue := range report.Issues {
			if strings.TrimSpace(issue.Hint) == "" {
				t.Errorf("scenario %d: issue for field %q has an empty hint", i, issue.Field)
			}
		}
	}
}

// requireProfileIssueField asserts that at least one issue targets the named field.
func requireProfileIssueField(t *testing.T, report *ProfileSessionReport, field string) {
	t.Helper()
	for _, issue := range report.Issues {
		if issue.Field == field {
			return
		}
	}
	t.Errorf("expected an issue for field %q, got: %v", field, report.Issues)
}
