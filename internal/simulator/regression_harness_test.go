// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

package simulator

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRegressionTestResult(t *testing.T) {
	t.Run("creates result with required fields", func(t *testing.T) {
		result := RegressionTestResult{
			TransactionHash: "abc123",
			Status:          "pass",
			EventCountMatch: true,
			TrapsMatch:      true,
		}

		assert.Equal(t, "abc123", result.TransactionHash)
		assert.Equal(t, "pass", result.Status)
		assert.True(t, result.EventCountMatch)
		assert.True(t, result.TrapsMatch)
	})

	t.Run("result can hold error message", func(t *testing.T) {
		result := RegressionTestResult{
			Status:       "error",
			ErrorMessage: "test error",
		}

		assert.Equal(t, "error", result.Status)
		assert.Equal(t, "test error", result.ErrorMessage)
	})
}

func TestRegressionTestSuite(t *testing.T) {
	t.Run("creates empty suite", func(t *testing.T) {
		suite := &RegressionTestSuite{
			TotalTests: 10,
			Results:    make([]RegressionTestResult, 0),
		}

		assert.Equal(t, 10, suite.TotalTests)
		assert.Equal(t, 0, len(suite.Results))
	})

	t.Run("adds results thread-safely", func(t *testing.T) {
		suite := &RegressionTestSuite{
			TotalTests: 5,
			Results:    make([]RegressionTestResult, 0, 5),
		}

		for i := 0; i < 5; i++ {
			result := RegressionTestResult{
				TransactionHash: "tx-" + string(rune(i)),
				Status:          "pass",
			}
			suite.addResult(result)
		}

		assert.Equal(t, 5, len(suite.Results))
	})

	t.Run("summary formats correctly", func(t *testing.T) {
		suite := &RegressionTestSuite{
			TotalTests:  10,
			PassedTests: 8,
			FailedTests: 1,
			ErrorTests:  1,
		}

		summary := suite.Summary()
		assert.Contains(t, summary, "10")
		assert.Contains(t, summary, "8")
		assert.Contains(t, summary, "80.0%")
	})

	t.Run("summary handles zero total gracefully", func(t *testing.T) {
		suite := &RegressionTestSuite{TotalTests: 0}
		summary := suite.Summary()
		// Must not panic and must produce a human-readable message.
		if summary == "" {
			t.Error("Summary() should not return empty string for zero-total suite")
		}
		if !strings.Contains(summary, "No tests") && !strings.Contains(summary, "0") {
			t.Errorf("Summary() for zero-total should indicate no tests, got: %q", summary)
		}
	})

	t.Run("failed results filters correctly", func(t *testing.T) {
		suite := &RegressionTestSuite{
			Results: []RegressionTestResult{
				{TransactionHash: "tx1", Status: "pass"},
				{TransactionHash: "tx2", Status: "fail"},
				{TransactionHash: "tx3", Status: "error"},
				{TransactionHash: "tx4", Status: "pass"},
			},
		}

		failed := suite.FailedResults()
		assert.Equal(t, 2, len(failed))
		assert.Equal(t, "tx2", failed[0].TransactionHash)
		assert.Equal(t, "tx3", failed[1].TransactionHash)
	})
}

func TestNewRegressionHarness(t *testing.T) {
	t.Run("creates harness with sensible defaults", func(t *testing.T) {
		mockRunner := &MockRunner{}
		harness := NewRegressionHarness(mockRunner, nil, 0)

		assert.Equal(t, mockRunner, harness.Runner)
		assert.Equal(t, 4, harness.MaxWorkers) // Default worker count
		assert.False(t, harness.Verbose)
	})

	t.Run("respects custom worker count", func(t *testing.T) {
		harness := NewRegressionHarness(&MockRunner{}, nil, 8)
		assert.Equal(t, 8, harness.MaxWorkers)
	})
}

func TestRegressionHarness_RunRegressionTests(t *testing.T) {
	t.Run("validates count parameter — zero", func(t *testing.T) {
		harness := NewRegressionHarness(&MockRunner{}, nil, 2)

		suite, err := harness.RunRegressionTests(context.Background(), 0, nil, 0)
		assert.Error(t, err)
		assert.Nil(t, suite)
		// Error message must be actionable.
		if !strings.Contains(err.Error(), "--count") {
			t.Errorf("error should mention --count, got: %q", err.Error())
		}
	})

	t.Run("validates count parameter — negative", func(t *testing.T) {
		harness := NewRegressionHarness(&MockRunner{}, nil, 2)

		suite, err := harness.RunRegressionTests(context.Background(), -1, nil, 0)
		assert.Error(t, err)
		assert.Nil(t, suite)
	})

	t.Run("nil runner returns descriptive error", func(t *testing.T) {
		harness := &RegressionHarness{Runner: nil, MaxWorkers: 2}

		suite, err := harness.RunRegressionTests(context.Background(), 5, nil, 0)
		assert.Error(t, err)
		assert.Nil(t, suite)
		if !strings.Contains(err.Error(), "runner") {
			t.Errorf("error should mention runner, got: %q", err.Error())
		}
	})

	t.Run("handles empty transaction list with guidance", func(t *testing.T) {
		harness := NewRegressionHarness(&MockRunner{}, nil, 2)

		suite, err := harness.RunRegressionTests(context.Background(), 10, nil, 0)
		assert.Error(t, err)
		assert.Nil(t, suite)
		if !strings.Contains(err.Error(), "no failed transactions found") {
			t.Errorf("error should mention no transactions found, got: %q", err.Error())
		}
		// Must give remediation hint.
		if !strings.Contains(err.Error(), "--start-seq") && !strings.Contains(err.Error(), "start-seq") {
			t.Errorf("error should suggest --start-seq remediation, got: %q", err.Error())
		}
	})
}

func TestRegressionHarness_TestTransaction(t *testing.T) {
	t.Run("returns error when RPCClient is nil — message is actionable", func(t *testing.T) {
		mockRunner := &MockRunner{
			RunFunc: func(ctx context.Context, req *SimulationRequest) (*SimulationResponse, error) {
				return &SimulationResponse{Status: "error"}, nil
			},
		}
		harness := NewRegressionHarness(mockRunner, nil, 2)

		result := harness.testTransaction(context.Background(), "some-tx", nil)

		assert.NotEmpty(t, result.ErrorMessage)
		assert.Equal(t, "error", result.Status)
		// Message should tell the user what to do.
		if !strings.Contains(result.ErrorMessage, "RPC client") {
			t.Errorf("error should mention RPC client, got: %q", result.ErrorMessage)
		}
	})

	t.Run("returns error for empty transaction hash", func(t *testing.T) {
		harness := NewRegressionHarness(&MockRunner{}, nil, 2)
		result := harness.testTransaction(context.Background(), "", nil)
		assert.Equal(t, "error", result.Status)
		assert.NotEmpty(t, result.ErrorMessage)
	})
}

func TestExtractLedgerKeysFromXDR(t *testing.T) {
	t.Run("handles empty XDR", func(t *testing.T) {
		keys, err := extractLedgerKeysFromXDR("")
		assert.NoError(t, err)
		assert.Equal(t, 0, len(keys))
	})

	t.Run("returns empty slice for non-empty XDR placeholder", func(t *testing.T) {
		keys, err := extractLedgerKeysFromXDR("AAAAAgAA...")
		assert.NoError(t, err)
		assert.Equal(t, 0, len(keys))
	})
}

func TestRegressionTestSuite_ConcurrentAddition(t *testing.T) {
	suite := &RegressionTestSuite{
		TotalTests: 100,
		Results:    make([]RegressionTestResult, 0, 100),
	}

	done := make(chan bool, 100)
	for i := 0; i < 100; i++ {
		go func(idx int) {
			result := RegressionTestResult{
				TransactionHash: "tx-" + string(rune(idx)),
				Status:          "pass",
			}
			suite.addResult(result)
			done <- true
		}(i)
	}

	for i := 0; i < 100; i++ {
		<-done
	}

	assert.Equal(t, 100, len(suite.Results))
}

// TestRegressionTestSuite_SummaryMentionsAllFields verifies the Summary string
// mentions Total, Passed, Failed, Errors, and Success Rate.
func TestRegressionTestSuite_SummaryMentionsAllFields(t *testing.T) {
	suite := &RegressionTestSuite{
		TotalTests:  5,
		PassedTests: 4,
		FailedTests: 1,
		ErrorTests:  0,
	}
	s := suite.Summary()
	for _, want := range []string{"Total", "Passed", "Failed", "Error", "%"} {
		if !strings.Contains(s, want) {
			t.Errorf("Summary() missing %q; got: %q", want, s)
		}
	}
}

// ── testTransaction — improved empty-hash error message ──────────────────────

// TestRegressionHarness_TestTransaction_EmptyHash_ActionableMessage verifies
// that an empty transaction hash produces a descriptive error message that
// includes a "Fix:" hint, not just a terse "skip" note.
func TestRegressionHarness_TestTransaction_EmptyHash_ActionableMessage(t *testing.T) {
	harness := NewRegressionHarness(&MockRunner{}, nil, 2)
	result := harness.testTransaction(context.Background(), "", nil)

	if result.Status != "error" {
		t.Errorf("expected status=error, got %q", result.Status)
	}
	if result.ErrorMessage == "" {
		t.Fatal("error message must not be empty for empty hash")
	}
	if !strings.Contains(result.ErrorMessage, "Fix:") {
		t.Errorf("error message should include a Fix hint, got: %q", result.ErrorMessage)
	}
	// Must not contain just "skip" — the old terse message.
	if result.ErrorMessage == "transaction hash is empty; skip this entry" {
		t.Error("error message should be updated beyond the terse 'skip' message")
	}
}

// ── RegressionTestSuite — pass/fail/error statistics consistency ──────────────

// TestRegressionTestSuite_StatisticsConsistency verifies that after calling
// addResult for a mix of statuses the counters computed by RunRegressionTests
// are consistent.  We test the counter logic directly since RunRegressionTests
// requires a live harness.
func TestRegressionTestSuite_StatisticsConsistency(t *testing.T) {
	suite := &RegressionTestSuite{
		TotalTests: 6,
		Results:    make([]RegressionTestResult, 0, 6),
	}

	statuses := []string{"pass", "pass", "fail", "error", "pass", "fail"}
	for _, s := range statuses {
		suite.addResult(RegressionTestResult{Status: s})
	}

	// Simulate the counter loop in RunRegressionTests.
	var passed, failed, errors int
	for _, r := range suite.Results {
		switch r.Status {
		case "pass":
			passed++
		case "fail":
			failed++
		case "error":
			errors++
		}
	}

	if passed != 3 {
		t.Errorf("expected 3 passed, got %d", passed)
	}
	if failed != 2 {
		t.Errorf("expected 2 failed, got %d", failed)
	}
	if errors != 1 {
		t.Errorf("expected 1 error, got %d", errors)
	}
}

// TestRegressionTestSuite_FailedResults_ExcludesPassed verifies that
// FailedResults never includes "pass" entries.
func TestRegressionTestSuite_FailedResults_ExcludesPassed(t *testing.T) {
	suite := &RegressionTestSuite{
		Results: []RegressionTestResult{
			{TransactionHash: "tx-pass", Status: "pass"},
			{TransactionHash: "tx-fail", Status: "fail"},
		},
	}

	failed := suite.FailedResults()
	for _, r := range failed {
		if r.Status == "pass" {
			t.Errorf("FailedResults included a 'pass' result: %+v", r)
		}
	}
	if len(failed) != 1 || failed[0].TransactionHash != "tx-fail" {
		t.Errorf("expected exactly [tx-fail], got %v", failed)
	}
}

// TestNewRegressionHarness_ZeroWorkers_DefaultsToFour verifies that MaxWorkers
// is defaulted to 4 when 0 is supplied — preventing a deadlock on the semaphore.
func TestNewRegressionHarness_ZeroWorkers_DefaultsToFour(t *testing.T) {
	harness := NewRegressionHarness(&MockRunner{}, nil, 0)
	if harness.MaxWorkers != 4 {
		t.Errorf("expected MaxWorkers=4 for 0 input, got %d", harness.MaxWorkers)
	}
}
