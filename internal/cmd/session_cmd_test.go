// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

// Tests for Issue #336: session persistence help output and examples.

package cmd

import (
	"strings"
	"testing"
)

// ── sessionCmd ────────────────────────────────────────────────────────────────

func TestSessionCmd_Long_ListsAllSubcommands(t *testing.T) {
	long := sessionCmd.Long
	for _, sub := range []string{"save", "resume", "list", "delete", "recover", "doctor"} {
		if !strings.Contains(long, sub) {
			t.Errorf("sessionCmd Long description should mention subcommand %q", sub)
		}
	}
}

func TestSessionCmd_Example_UsesLowercaseBinary(t *testing.T) {
	example := sessionCmd.Example
	if strings.Contains(example, "Glassbox ") {
		t.Error("sessionCmd Example should use lowercase 'glassbox', not 'Glassbox'")
	}
}

func TestSessionCmd_Example_MentionsRecover(t *testing.T) {
	if !strings.Contains(sessionCmd.Example, "recover") {
		t.Error("sessionCmd Example should mention 'recover' subcommand")
	}
}

func TestSessionCmd_Example_NonEmpty(t *testing.T) {
	if strings.TrimSpace(sessionCmd.Example) == "" {
		t.Error("sessionCmd must have a non-empty Example field")
	}
}

// ── sessionSaveCmd ────────────────────────────────────────────────────────────

func TestSessionSaveCmd_Example_UsesLowercaseBinary(t *testing.T) {
	if strings.Contains(sessionSaveCmd.Example, "Glassbox ") {
		t.Error("sessionSaveCmd Example should use lowercase 'glassbox', not 'Glassbox'")
	}
}

func TestSessionSaveCmd_Example_MentionsPinEndpoint(t *testing.T) {
	if !strings.Contains(sessionSaveCmd.Example, "--pin-endpoint") {
		t.Error("sessionSaveCmd Example should demonstrate --pin-endpoint flag")
	}
}

func TestSessionSaveCmd_Long_MentionsValidation(t *testing.T) {
	if !strings.Contains(sessionSaveCmd.Long, "Validation") && !strings.Contains(sessionSaveCmd.Long, "validation") {
		t.Error("sessionSaveCmd Long description should mention validation behavior")
	}
}

func TestSessionSaveCmd_Example_MentionsName(t *testing.T) {
	if !strings.Contains(sessionSaveCmd.Example, "--name") {
		t.Error("sessionSaveCmd Example should demonstrate --name flag")
	}
}

// ── sessionResumeCmd ──────────────────────────────────────────────────────────

func TestSessionResumeCmd_Example_UsesLowercaseBinary(t *testing.T) {
	if strings.Contains(sessionResumeCmd.Example, "Glassbox ") {
		t.Error("sessionResumeCmd Example should use lowercase 'glassbox', not 'Glassbox'")
	}
}

func TestSessionResumeCmd_Long_MentionsIntegrityCheck(t *testing.T) {
	long := sessionResumeCmd.Long
	if !strings.Contains(long, "integrity") {
		t.Error("sessionResumeCmd Long should mention the integrity check performed on resume")
	}
}

func TestSessionResumeCmd_Example_MentionsList(t *testing.T) {
	if !strings.Contains(sessionResumeCmd.Example, "list") {
		t.Error("sessionResumeCmd Example should hint at using 'session list' first")
	}
}

// ── sessionListCmd ────────────────────────────────────────────────────────────

func TestSessionListCmd_Example_UsesLowercaseBinary(t *testing.T) {
	if strings.Contains(sessionListCmd.Example, "Glassbox ") {
		t.Error("sessionListCmd Example should use lowercase 'glassbox', not 'Glassbox'")
	}
}

func TestSessionListCmd_Example_NonEmpty(t *testing.T) {
	if strings.TrimSpace(sessionListCmd.Example) == "" {
		t.Error("sessionListCmd must have a non-empty Example field")
	}
}

// ── sessionDeleteCmd ──────────────────────────────────────────────────────────

func TestSessionDeleteCmd_Example_UsesLowercaseBinary(t *testing.T) {
	if strings.Contains(sessionDeleteCmd.Example, "Glassbox ") {
		t.Error("sessionDeleteCmd Example should use lowercase 'glassbox', not 'Glassbox'")
	}
}

func TestSessionDeleteCmd_Example_MentionsList(t *testing.T) {
	if !strings.Contains(sessionDeleteCmd.Example, "list") {
		t.Error("sessionDeleteCmd Example should hint at using 'session list' to find the ID")
	}
}

// ── sessionRecoverCmd ─────────────────────────────────────────────────────────

func TestSessionRecoverCmd_Long_MentionsCheckpoint(t *testing.T) {
	long := sessionRecoverCmd.Long
	if !strings.Contains(long, "checkpoint") {
		t.Error("sessionRecoverCmd Long should explain checkpoint-based recovery")
	}
}

func TestSessionRecoverCmd_Long_MentionsValidation(t *testing.T) {
	long := sessionRecoverCmd.Long
	if !strings.Contains(long, "validation") && !strings.Contains(long, "Validation") {
		t.Error("sessionRecoverCmd Long should describe the validation performed on the checkpoint")
	}
}

func TestSessionRecoverCmd_Example_MentionsDebug(t *testing.T) {
	if !strings.Contains(sessionRecoverCmd.Example, "debug") {
		t.Error("sessionRecoverCmd Example should show how to re-run debug if recovery finds nothing")
	}
}

// ── sessionDoctorCmd ──────────────────────────────────────────────────────────

func TestSessionDoctorCmd_Example_MentionsDebug(t *testing.T) {
	if !strings.Contains(sessionDoctorCmd.Example, "debug") {
		t.Error("sessionDoctorCmd Example should suggest re-running debug to fix degraded sessions")
	}
}

func TestSessionDoctorCmd_Long_MentionsSchemaVersion(t *testing.T) {
	long := sessionDoctorCmd.Long
	if !strings.Contains(long, "schema") {
		t.Error("sessionDoctorCmd Long should mention schema version checks")
	}
}

// ── all subcommands have Short descriptions ───────────────────────────────────

func TestAllSessionSubcmds_HaveShortDescription(t *testing.T) {
	cmds := []*struct {
		name string
		short string
	}{
		{"save", sessionSaveCmd.Short},
		{"resume", sessionResumeCmd.Short},
		{"list", sessionListCmd.Short},
		{"delete", sessionDeleteCmd.Short},
		{"recover", sessionRecoverCmd.Short},
		{"doctor", sessionDoctorCmd.Short},
	}
	for _, c := range cmds {
		if strings.TrimSpace(c.short) == "" {
			t.Errorf("session %s subcommand must have a non-empty Short description", c.name)
		}
	}
}
