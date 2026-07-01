// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

package session

import (
	"strings"
	"testing"
	"time"
)

// validCheckpoint returns a fully populated Checkpoint that passes all validation.
func validCheckpoint() *Checkpoint {
	return &Checkpoint{
		SessionID: "session-abc123",
		TxHash:    "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		Network:   "testnet",
		StartedAt: time.Now().Add(-5 * time.Minute),
		PID:       12345,
	}
}

func TestValidateCheckpoint_ValidCheckpoint_OK(t *testing.T) {
	report := ValidateCheckpoint(validCheckpoint())
	if !report.OK {
		t.Errorf("expected OK=true for valid checkpoint, got issues: %v", report.Issues)
	}
	if len(report.Issues) != 0 {
		t.Errorf("expected zero issues, got %d: %v", len(report.Issues), report.Issues)
	}
}

func TestValidateCheckpoint_MissingSessionID_ReportsIssue(t *testing.T) {
	cp := validCheckpoint()
	cp.SessionID = ""
	report := ValidateCheckpoint(cp)
	if report.OK {
		t.Fatal("expected OK=false when SessionID is empty")
	}
	requireCheckpointIssueField(t, report, "SessionID")
}

func TestValidateCheckpoint_MissingTxHash_ReportsIssue(t *testing.T) {
	cp := validCheckpoint()
	cp.TxHash = ""
	report := ValidateCheckpoint(cp)
	if report.OK {
		t.Fatal("expected OK=false when TxHash is empty")
	}
	requireCheckpointIssueField(t, report, "TxHash")
}

func TestValidateCheckpoint_MissingNetwork_ReportsIssue(t *testing.T) {
	cp := validCheckpoint()
	cp.Network = ""
	report := ValidateCheckpoint(cp)
	if report.OK {
		t.Fatal("expected OK=false when Network is empty")
	}
	requireCheckpointIssueField(t, report, "Network")
}

func TestValidateCheckpoint_InvalidNetwork_ReportsIssue(t *testing.T) {
	cp := validCheckpoint()
	cp.Network = "devnet"
	report := ValidateCheckpoint(cp)
	if report.OK {
		t.Fatal("expected OK=false for unrecognised network")
	}
	requireCheckpointIssueField(t, report, "Network")
	for _, issue := range report.Issues {
		if issue.Field == "Network" && !strings.Contains(issue.Description, "devnet") {
			t.Errorf("issue description should name the bad value 'devnet', got: %q", issue.Description)
		}
	}
}

func TestValidateCheckpoint_ValidNetworks_OK(t *testing.T) {
	for _, net := range []string{"testnet", "mainnet", "futurenet"} {
		cp := validCheckpoint()
		cp.Network = net
		report := ValidateCheckpoint(cp)
		for _, issue := range report.Issues {
			if issue.Field == "Network" {
				t.Errorf("valid network %q should not produce a Network issue, got: %v", net, issue)
			}
		}
	}
}

func TestValidateCheckpoint_ZeroStartedAt_ReportsIssue(t *testing.T) {
	cp := validCheckpoint()
	cp.StartedAt = time.Time{}
	report := ValidateCheckpoint(cp)
	if report.OK {
		t.Fatal("expected OK=false when StartedAt is zero")
	}
	requireCheckpointIssueField(t, report, "StartedAt")
}

func TestValidateCheckpoint_ZeroPID_ReportsIssue(t *testing.T) {
	cp := validCheckpoint()
	cp.PID = 0
	report := ValidateCheckpoint(cp)
	if report.OK {
		t.Fatal("expected OK=false when PID is 0")
	}
	requireCheckpointIssueField(t, report, "PID")
}

func TestValidateCheckpoint_NegativePID_ReportsIssue(t *testing.T) {
	cp := validCheckpoint()
	cp.PID = -1
	report := ValidateCheckpoint(cp)
	if report.OK {
		t.Fatal("expected OK=false when PID is negative")
	}
	requireCheckpointIssueField(t, report, "PID")
}

func TestValidateCheckpoint_AllIssuesHaveHints(t *testing.T) {
	scenarios := []*Checkpoint{
		func() *Checkpoint { cp := validCheckpoint(); cp.SessionID = ""; return cp }(),
		func() *Checkpoint { cp := validCheckpoint(); cp.TxHash = ""; return cp }(),
		func() *Checkpoint { cp := validCheckpoint(); cp.Network = ""; return cp }(),
		func() *Checkpoint { cp := validCheckpoint(); cp.Network = "badnet"; return cp }(),
		func() *Checkpoint { cp := validCheckpoint(); cp.StartedAt = time.Time{}; return cp }(),
		func() *Checkpoint { cp := validCheckpoint(); cp.PID = 0; return cp }(),
	}
	for i, cp := range scenarios {
		report := ValidateCheckpoint(cp)
		for _, issue := range report.Issues {
			if strings.TrimSpace(issue.Hint) == "" {
				t.Errorf("scenario %d: issue for field %q has an empty hint", i, issue.Field)
			}
		}
	}
}

func TestValidateCheckpoint_MultipleProblems_AllReported(t *testing.T) {
	cp := &Checkpoint{} // all zero values
	report := ValidateCheckpoint(cp)
	if len(report.Issues) < 4 {
		t.Errorf("expected at least 4 issues for empty checkpoint, got %d: %v", len(report.Issues), report.Issues)
	}
}

// requireCheckpointIssueField asserts that at least one issue targets the named field.
func requireCheckpointIssueField(t *testing.T, report *CheckpointReport, field string) {
	t.Helper()
	for _, issue := range report.Issues {
		if issue.Field == field {
			return
		}
	}
	t.Errorf("expected an issue for field %q, got: %v", field, report.Issues)
}
