// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

package analyzer

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSecurityBoundaryChecker_NoViolations(t *testing.T) {
	events := []string{
		`{"type":"contract_call","contract":"CABC123"}`,
		`{"type":"auth","address":"GABC123","contract":"CABC123"}`,
		`{"type":"storage_write","contract":"CABC123"}`,
	}

	checker := NewSecurityBoundaryChecker()
	violations, err := checker.Analyze(events)

	assert.NoError(t, err)
	assert.Empty(t, violations)
}

func TestSecurityBoundaryChecker_UnauthorizedStateModification(t *testing.T) {
	events := []string{
		`{"type":"contract_call","contract":"CABC123"}`,
		`{"type":"storage_write","contract":"CABC123"}`,
	}

	checker := NewSecurityBoundaryChecker()
	violations, err := checker.Analyze(events)

	assert.NoError(t, err)
	assert.Len(t, violations, 1)
	assert.Equal(t, "unauthorized_state_modification", violations[0].Type)
	assert.Equal(t, "high", violations[0].Severity)
	assert.Equal(t, "CABC123", violations[0].Contract)
}

func TestSecurityBoundaryChecker_SACPattern_NoFalsePositive(t *testing.T) {
	testCases := []struct {
		name     string
		contract string
	}{
		{"SAC balance update", "SAC_token_contract"},
		{"SAC allowance update", "stellar_asset_contract"},
		{"SAC admin operation", "token_SAC"},
		{"Stellar asset contract", "stellar_asset"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			events := []string{
				fmt.Sprintf(`{"type":"storage_write","contract":"%s"}`, tc.contract),
			}

			checker := NewSecurityBoundaryChecker()
			violations, err := checker.Analyze(events)

			assert.NoError(t, err)
			assert.Empty(t, violations, "SAC pattern should not trigger violations")
		})
	}
}

func TestSecurityBoundaryChecker_MultipleContracts(t *testing.T) {
	events := []string{
		`{"type":"contract_call","contract":"C1"}`,
		`{"type":"auth","address":"A1","contract":"C1"}`,
		`{"type":"storage_write","contract":"C1"}`,
		`{"type":"contract_call","contract":"C2"}`,
		`{"type":"storage_write","contract":"C2"}`,
	}

	checker := NewSecurityBoundaryChecker()
	violations, err := checker.Analyze(events)

	assert.NoError(t, err)
	assert.Len(t, violations, 1)
	assert.Equal(t, "C2", violations[0].Contract)
}

func TestSecurityBoundaryChecker_AuthAfterWrite_StillViolation(t *testing.T) {
	events := []string{
		`{"type":"contract_call","contract":"CABC"}`,
		`{"type":"storage_write","contract":"CABC"}`,
		`{"type":"auth","address":"GABC","contract":"CABC"}`,
	}

	checker := NewSecurityBoundaryChecker()
	violations, err := checker.Analyze(events)

	assert.NoError(t, err)
	assert.Len(t, violations, 1)
}

func TestSecurityBoundaryChecker_InvalidJSON_Skipped(t *testing.T) {
	events := []string{
		`invalid json`,
		`{"type":"contract_call","contract":"CABC"}`,
		`{"type":"storage_write","contract":"CABC"}`,
	}

	checker := NewSecurityBoundaryChecker()
	violations, err := checker.Analyze(events)

	assert.NoError(t, err)
	assert.Len(t, violations, 1)
}

func TestSecurityBoundaryChecker_EmptyEvents(t *testing.T) {
	events := []string{}

	checker := NewSecurityBoundaryChecker()
	violations, err := checker.Analyze(events)

	assert.NoError(t, err)
	assert.Empty(t, violations)
}

func TestSecurityBoundaryChecker_UnknownContract_Skipped(t *testing.T) {
	events := []string{
		`{"type":"storage_write","contract":"unknown"}`,
		`{"type":"storage_write","contract":""}`,
	}

	checker := NewSecurityBoundaryChecker()
	violations, err := checker.Analyze(events)

	assert.NoError(t, err)
	assert.Empty(t, violations)
}

// TestSecurityBoundaryChecker_ADR003ReviewScenarios exercises the trust-boundary
// review scenarios documented in docs/adr/003-trust-boundaries.md. It mirrors the
// documented command:
//
//	go test ./internal/analyzer -run TestSecurityBoundaryChecker -v
//
// The success path shows an auth event preceding a storage_write on a non-SAC
// contract producing no violations. The failure path shows a storage_write
// without prior auth on a non-SAC contract producing exactly one
// unauthorized_state_modification violation at high severity.
func TestSecurityBoundaryChecker_ADR003ReviewScenarios(t *testing.T) {
	t.Run("success: auth precedes storage_write", func(t *testing.T) {
		events := []string{
			`{"type":"contract_call","contract":"CABC123"}`,
			`{"type":"auth","address":"GABC123","contract":"CABC123"}`,
			`{"type":"storage_write","contract":"CABC123"}`,
		}

		checker := NewSecurityBoundaryChecker()
		violations, err := checker.Analyze(events)

		assert.NoError(t, err)
		assert.Empty(t, violations)
	})

	t.Run("failure: storage_write without prior auth", func(t *testing.T) {
		events := []string{
			`{"type":"contract_call","contract":"CABC123"}`,
			`{"type":"storage_write","contract":"CABC123"}`,
		}

		checker := NewSecurityBoundaryChecker()
		violations, err := checker.Analyze(events)

		assert.NoError(t, err)
		assert.Len(t, violations, 1)
		assert.Equal(t, "unauthorized_state_modification", violations[0].Type)
		assert.Equal(t, "high", violations[0].Severity)
		assert.Equal(t, "CABC123", violations[0].Contract)
	})
}
