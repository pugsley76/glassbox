// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

// Tests for Part C: debug command regression and mock harness improvements.
// Covers validateRegressionFlags and the regression-test command's validation
// paths to ensure early, actionable error messages.

package cmd

import (
	"fmt"
	"strings"
	"testing"
)

// resetRegressionFlags restores all regression-test flag variables to their
// defaults so each test starts from a clean state.
func resetRegressionFlags() {
	regressionTestCount = 100
	regressionProtocolVersion = 0
	regressionStartSeq = 0
	regressionMaxWorkers = 4
	networkFlag = "mainnet"
	rpcURLFlag = ""
	rpcTokenFlag = ""
}

// ── --count validation ────────────────────────────────────────────────────────

// TestValidateRegressionFlags_ZeroCount verifies that --count=0 is rejected
// with an explicit message explaining the valid range.
func TestValidateRegressionFlags_ZeroCount(t *testing.T) {
	t.Cleanup(resetRegressionFlags)
	regressionTestCount = 0

	err := validateRegressionFlags(regressionTestCmd, []string{})
	if err == nil {
		t.Fatal("expected error for --count=0")
	}
	msg := err.Error()
	if !strings.Contains(msg, "--count") {
		t.Errorf("error should mention --count, got: %q", msg)
	}
	if !strings.Contains(msg, "greater than 0") {
		t.Errorf("error should state count must be > 0, got: %q", msg)
	}
}

// TestValidateRegressionFlags_NegativeCount verifies that --count=-1 is rejected.
func TestValidateRegressionFlags_NegativeCount(t *testing.T) {
	t.Cleanup(resetRegressionFlags)
	regressionTestCount = -1

	err := validateRegressionFlags(regressionTestCmd, []string{})
	if err == nil {
		t.Fatal("expected error for --count=-1")
	}
	if !strings.Contains(err.Error(), "--count") {
		t.Errorf("error should mention --count, got: %q", err.Error())
	}
}

// TestValidateRegressionFlags_CountTooLarge verifies that --count over the max
// is rejected with a message naming the limit.
func TestValidateRegressionFlags_CountTooLarge(t *testing.T) {
	t.Cleanup(resetRegressionFlags)
	regressionTestCount = maxRegressionCount + 1

	err := validateRegressionFlags(regressionTestCmd, []string{})
	if err == nil {
		t.Fatal("expected error for --count exceeding maximum")
	}
	msg := err.Error()
	if !strings.Contains(msg, "exceed") && !strings.Contains(msg, "maximum") {
		t.Errorf("error should mention maximum limit, got: %q", msg)
	}
}

// TestValidateRegressionFlags_ValidCount verifies that a valid --count passes.
func TestValidateRegressionFlags_ValidCount(t *testing.T) {
	t.Cleanup(resetRegressionFlags)
	regressionTestCount = 50
	networkFlag = "mainnet"

	err := validateRegressionFlags(regressionTestCmd, []string{})
	if err != nil {
		t.Errorf("validateRegressionFlags() should accept --count=50, got: %v", err)
	}
}

// TestValidateRegressionFlags_BoundaryCount verifies that --count=1 and
// --count=1000 (the boundaries) are both accepted.
func TestValidateRegressionFlags_BoundaryCount(t *testing.T) {
	for _, c := range []int{1, maxRegressionCount} {
		c := c
		t.Run(fmt.Sprintf("count_%d", c), func(t *testing.T) {
			t.Cleanup(resetRegressionFlags)
			regressionTestCount = c
			networkFlag = "mainnet"

			err := validateRegressionFlags(regressionTestCmd, []string{})
			if err != nil && strings.Contains(err.Error(), "--count") {
				t.Errorf("count %d should be valid, got: %v", c, err)
			}
		})
	}
}

// ── --workers validation ──────────────────────────────────────────────────────

// TestValidateRegressionFlags_NegativeWorkers verifies that --workers=-1 is
// rejected before any network calls.
func TestValidateRegressionFlags_NegativeWorkers(t *testing.T) {
	t.Cleanup(resetRegressionFlags)
	regressionMaxWorkers = -1

	err := validateRegressionFlags(regressionTestCmd, []string{})
	if err == nil {
		t.Fatal("expected error for --workers=-1")
	}
	if !strings.Contains(err.Error(), "--workers") {
		t.Errorf("error should mention --workers, got: %q", err.Error())
	}
}

// ── --network validation ──────────────────────────────────────────────────────

// TestValidateRegressionFlags_InvalidNetwork verifies an unknown --network value
// is rejected early with a clear message listing valid options.
func TestValidateRegressionFlags_InvalidNetwork(t *testing.T) {
	t.Cleanup(resetRegressionFlags)
	networkFlag = "prodnet" // invalid

	err := validateRegressionFlags(regressionTestCmd, []string{})
	if err == nil {
		t.Fatal("expected error for invalid --network")
	}
	msg := err.Error()
	if !strings.Contains(msg, "prodnet") {
		t.Errorf("error should echo the invalid value, got: %q", msg)
	}
	if !strings.Contains(msg, "testnet") {
		t.Errorf("error should list valid networks, got: %q", msg)
	}
}

// TestValidateRegressionFlags_ValidNetworks verifies testnet, mainnet, and
// futurenet all pass network validation.
func TestValidateRegressionFlags_ValidNetworks(t *testing.T) {
	for _, n := range []string{"testnet", "mainnet", "futurenet"} {
		n := n
		t.Run(n, func(t *testing.T) {
			t.Cleanup(resetRegressionFlags)
			networkFlag = n

			err := validateRegressionFlags(regressionTestCmd, []string{})
			if err != nil && strings.Contains(err.Error(), "--network") {
				t.Errorf("network %q should be valid, got: %v", n, err)
			}
		})
	}
}

// ── command Example field ─────────────────────────────────────────────────────

// TestRegressionTestCmd_ExamplePresent verifies the regression-test command has
// a non-empty Example field so --help output is helpful.
func TestRegressionTestCmd_ExamplePresent(t *testing.T) {
	if strings.TrimSpace(regressionTestCmd.Example) == "" {
		t.Error("regression-test command must have a non-empty Example field")
	}
}

// TestRegressionTestCmd_ExampleContainsCount verifies the Example mentions --count.
func TestRegressionTestCmd_ExampleContainsCount(t *testing.T) {
	if !strings.Contains(regressionTestCmd.Example, "--count") {
		t.Error("regression-test Example should reference --count flag")
	}
}

// TestRegressionTestCmd_ExampleContainsWorkers verifies the Example mentions --workers.
func TestRegressionTestCmd_ExampleContainsWorkers(t *testing.T) {
	if !strings.Contains(regressionTestCmd.Example, "--workers") {
		t.Error("regression-test Example should reference --workers flag")
	}
}

// ── help text quality ─────────────────────────────────────────────────────────

// TestRegressionTestCmd_LongDescriptionMentionsValidation verifies the Long
// description tells users what --count, --workers, and --network accept.
func TestRegressionTestCmd_LongDescriptionMentionsValidation(t *testing.T) {
	long := regressionTestCmd.Long
	if !strings.Contains(long, "--count") {
		t.Error("Long description should mention --count validation")
	}
	if !strings.Contains(long, "--network") {
		t.Error("Long description should mention --network validation")
	}
}

// ── --workers=0 auto-correction behavior ─────────────────────────────────────

// TestValidateRegressionFlags_ZeroWorkers_Passes verifies that --workers=0
// passes PreRunE (it is auto-corrected to 4 in RunE, not rejected upfront).
func TestValidateRegressionFlags_ZeroWorkers_Passes(t *testing.T) {
	t.Cleanup(resetRegressionFlags)
	regressionMaxWorkers = 0
	networkFlag = "mainnet"

	err := validateRegressionFlags(regressionTestCmd, []string{})
	if err != nil && strings.Contains(err.Error(), "--workers") {
		t.Errorf("--workers=0 should pass PreRunE (auto-corrected to 4), got: %v", err)
	}
}

// TestValidateRegressionFlags_NegativeWorkers_IsRejected verifies that a
// genuinely negative --workers value is caught before any RPC calls.
func TestValidateRegressionFlags_NegativeWorkers_IsRejected(t *testing.T) {
	t.Cleanup(resetRegressionFlags)
	regressionMaxWorkers = -5
	networkFlag = "mainnet"

	err := validateRegressionFlags(regressionTestCmd, []string{})
	if err == nil {
		t.Fatal("expected error for --workers=-5")
	}
	if !strings.Contains(err.Error(), "--workers") {
		t.Errorf("error should mention --workers, got: %q", err.Error())
	}
}

// ── Long description accuracy ─────────────────────────────────────────────────

// TestRegressionTestCmd_LongDescriptionMentionsWorkers verifies the Long
// description references --workers so users know it can be configured.
func TestRegressionTestCmd_LongDescriptionMentionsWorkers(t *testing.T) {
	if !strings.Contains(regressionTestCmd.Long, "--workers") {
		t.Error("Long description should mention --workers validation behavior")
	}
}

// TestRegressionTestCmd_LongDescriptionAccurateForZeroWorkers verifies that
// the Long description correctly states workers=0 defaults to 4 (not that it
// must be "a positive integer").
func TestRegressionTestCmd_LongDescriptionAccurateForZeroWorkers(t *testing.T) {
	long := regressionTestCmd.Long
	// The old (incorrect) wording was "must be a positive integer".
	// The correct wording accepts 0 with an auto-correction note.
	if strings.Contains(long, "must be a positive integer") {
		t.Error("Long description incorrectly says workers must be a positive integer; " +
			"0 is allowed (auto-corrected to 4)")
	}
}

// ── --count boundary enforcement ─────────────────────────────────────────────

// TestValidateRegressionFlags_CountAtExactMaximum verifies count=1000 is accepted.
func TestValidateRegressionFlags_CountAtExactMaximum(t *testing.T) {
	t.Cleanup(resetRegressionFlags)
	regressionTestCount = maxRegressionCount // 1000
	networkFlag = "mainnet"

	err := validateRegressionFlags(regressionTestCmd, []string{})
	if err != nil && strings.Contains(err.Error(), "--count") {
		t.Errorf("count=%d (exact maximum) should be accepted, got: %v", maxRegressionCount, err)
	}
}

// TestValidateRegressionFlags_CountOneOverMaximum verifies count=1001 is rejected.
func TestValidateRegressionFlags_CountOneOverMaximum(t *testing.T) {
	t.Cleanup(resetRegressionFlags)
	regressionTestCount = maxRegressionCount + 1
	networkFlag = "mainnet"

	err := validateRegressionFlags(regressionTestCmd, []string{})
	if err == nil {
		t.Fatalf("expected error for count=%d (one over maximum)", maxRegressionCount+1)
	}
	if !strings.Contains(err.Error(), "exceed") && !strings.Contains(err.Error(), "maximum") {
		t.Errorf("error should mention exceeding maximum, got: %q", err.Error())
	}
	// Error must include the actual count for clarity.
	if !strings.Contains(err.Error(), fmt.Sprintf("%d", maxRegressionCount+1)) {
		t.Errorf("error should include the supplied count value, got: %q", err.Error())
	}
}
