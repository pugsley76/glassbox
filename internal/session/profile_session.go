// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

package session

import (
	"fmt"
	"strings"
	"time"
)

// SupportedProfileFormats is the set of output formats accepted by the profile command.
var SupportedProfileFormats = []string{"html", "svg", "pprof"}

// ProfileSession records the inputs and outputs of a profile (gas analysis)
// run so it can be associated with the parent debug session and replayed.
type ProfileSession struct {
	// SessionID links this profile to the parent debug session.
	SessionID string `json:"session_id"`
	// TxHash is the transaction that was profiled.
	TxHash string `json:"tx_hash"`
	// Network is the Stellar network the transaction ran on.
	Network string `json:"network"`
	// Format is the output format (html, svg, pprof).
	Format string `json:"format"`
	// OutputPath is where the generated flamegraph or pprof file was written.
	OutputPath string `json:"output_path,omitempty"`
	// StartedAt is when the profile run began.
	StartedAt time.Time `json:"started_at"`
	// CompletedAt is when the profile run finished (zero if not yet done).
	CompletedAt time.Time `json:"completed_at,omitempty"`
	// Error holds the error message if the run failed.
	Error string `json:"error,omitempty"`
}

// ProfileSessionIssue describes a single validation problem in a ProfileSession.
type ProfileSessionIssue struct {
	Field       string
	Description string
	Hint        string
}

// ProfileSessionReport is returned by ValidateProfileSession.
type ProfileSessionReport struct {
	OK     bool
	Issues []ProfileSessionIssue
}

// ValidateProfileSession checks a ProfileSession record for completeness and
// consistency before it is persisted or replayed. It validates:
//
//   - SessionID is non-empty
//   - TxHash is non-empty
//   - Network is a recognised Stellar network value
//   - Format is one of the supported output formats
//   - StartedAt is non-zero
//   - CompletedAt, when set, is not before StartedAt
//
// The function never modifies the session and is safe to call concurrently.
func ValidateProfileSession(ps *ProfileSession) *ProfileSessionReport {
	report := &ProfileSessionReport{}

	if ps.SessionID == "" {
		report.Issues = append(report.Issues, ProfileSessionIssue{
			Field:       "SessionID",
			Description: "profile session is missing the parent session ID",
			Hint:        "Run 'glassbox debug <tx-hash>' first to create a session, then profile it.",
		})
	}

	if ps.TxHash == "" {
		report.Issues = append(report.Issues, ProfileSessionIssue{
			Field:       "TxHash",
			Description: "profile session is missing the transaction hash",
			Hint:        "Re-run 'glassbox profile --xdr <tx.xdr>' with a valid transaction envelope.",
		})
	}

	if ps.Network == "" {
		report.Issues = append(report.Issues, ProfileSessionIssue{
			Field:       "Network",
			Description: "profile session is missing the network",
			Hint:        "Specify the network with --network testnet|mainnet|futurenet.",
		})
	} else {
		validNetworks := map[string]bool{
			"testnet": true, "mainnet": true, "futurenet": true,
		}
		if !validNetworks[strings.ToLower(ps.Network)] {
			report.Issues = append(report.Issues, ProfileSessionIssue{
				Field:       "Network",
				Description: fmt.Sprintf("profile session network %q is not a recognised Stellar network", ps.Network),
				Hint:        "Accepted values are: testnet, mainnet, futurenet.",
			})
		}
	}

	if ps.Format == "" {
		report.Issues = append(report.Issues, ProfileSessionIssue{
			Field:       "Format",
			Description: "profile session is missing the output format",
			Hint:        fmt.Sprintf("Specify a format with --profile-format. Accepted values: %s.", strings.Join(SupportedProfileFormats, ", ")),
		})
	} else {
		validFormats := map[string]bool{}
		for _, f := range SupportedProfileFormats {
			validFormats[f] = true
		}
		if !validFormats[strings.ToLower(ps.Format)] {
			report.Issues = append(report.Issues, ProfileSessionIssue{
				Field:       "Format",
				Description: fmt.Sprintf("profile format %q is not supported", ps.Format),
				Hint:        fmt.Sprintf("Accepted values are: %s.", strings.Join(SupportedProfileFormats, ", ")),
			})
		}
	}

	if ps.StartedAt.IsZero() {
		report.Issues = append(report.Issues, ProfileSessionIssue{
			Field:       "StartedAt",
			Description: "profile session has a zero started_at timestamp",
			Hint:        "This is a data-integrity issue; delete the profile session record and re-run 'glassbox profile'.",
		})
	}

	if !ps.CompletedAt.IsZero() && !ps.StartedAt.IsZero() {
		if ps.CompletedAt.Before(ps.StartedAt) {
			report.Issues = append(report.Issues, ProfileSessionIssue{
				Field:       "CompletedAt",
				Description: "completed_at is before started_at — timestamps are inconsistent",
				Hint:        "The profile session record is corrupt; delete it and re-run 'glassbox profile'.",
			})
		}
	}

	report.OK = len(report.Issues) == 0
	return report
}
