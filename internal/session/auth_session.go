// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

package session

import (
	"encoding/hex"
	"fmt"
	"net/url"
	"strings"
	"time"
)

// txHashLength is the exact number of hexadecimal characters in a Stellar
// transaction hash. It is duplicated here (rather than importing internal/rpc)
// to keep the session package free of a dependency on the RPC layer, matching
// the self-contained style of ProfileSession validation.
const txHashLength = 64

// SupportedAuthSessionStatuses is the set of terminal states an auth-debug run
// can be persisted with. Any other value is rejected by ValidateAuthSession.
//
//   - completed:    authorization data was extracted and analyzed.
//   - failed:       the run errored before producing a usable analysis.
//   - no_auth_data: the transaction carried no Soroban authorization entries,
//     so the report reflects "no failures recorded", not a verified pass.
var SupportedAuthSessionStatuses = []string{"completed", "failed", "no_auth_data"}

// AuthSession records the inputs and outcome of an auth-debug run so it can be
// associated with the parent debug session and replayed or audited later. It is
// the persistence counterpart of the in-memory authorization trace produced by
// the auth-debug command.
type AuthSession struct {
	// SessionID links this auth analysis to the parent debug session.
	SessionID string `json:"session_id"`
	// TxHash is the transaction whose authorization flow was analyzed.
	TxHash string `json:"tx_hash"`
	// Network is the Stellar network the transaction ran on.
	Network string `json:"network"`
	// RPCURL is the custom Horizon/RPC endpoint used, if any.
	RPCURL string `json:"rpc_url,omitempty"`
	// Detailed records whether --detailed analysis was requested.
	Detailed bool `json:"detailed,omitempty"`
	// JSONOutput records whether --json output was requested.
	JSONOutput bool `json:"json_output,omitempty"`
	// AuthEventCount is the number of authorization events extracted.
	AuthEventCount int `json:"auth_event_count"`
	// FailureCount is the number of authorization failures recorded.
	FailureCount int `json:"failure_count"`
	// MissingSignatureCount is the number of required signers still missing.
	MissingSignatureCount int `json:"missing_signature_count"`
	// Status is the terminal state of the run (see SupportedAuthSessionStatuses).
	Status string `json:"status"`
	// StartedAt is when the auth-debug run began.
	StartedAt time.Time `json:"started_at"`
	// CompletedAt is when the run finished (zero if not yet done).
	CompletedAt time.Time `json:"completed_at,omitempty"`
	// Error holds the error message if the run failed.
	Error string `json:"error,omitempty"`
}

// AuthSessionIssue describes a single validation problem in an AuthSession.
type AuthSessionIssue struct {
	Field       string
	Description string
	Hint        string
}

// AuthSessionReport is returned by ValidateAuthSession.
type AuthSessionReport struct {
	OK     bool
	Issues []AuthSessionIssue
}

// Error renders the report as a single actionable, multi-line error message, or
// returns nil when the report is OK. Each issue is printed with its remediation
// hint so failures are surfaced clearly rather than as a bare "invalid" string.
func (r *AuthSessionReport) Error() error {
	if r.OK || len(r.Issues) == 0 {
		return nil
	}

	var sb strings.Builder
	if len(r.Issues) == 1 {
		sb.WriteString("auth session is invalid:")
	} else {
		fmt.Fprintf(&sb, "auth session is invalid (%d problems):", len(r.Issues))
	}
	for _, issue := range r.Issues {
		fmt.Fprintf(&sb, "\n  - %s: %s", issue.Field, issue.Description)
		if issue.Hint != "" {
			fmt.Fprintf(&sb, "\n    Hint: %s", issue.Hint)
		}
	}
	return fmt.Errorf("%s", sb.String())
}

// ValidateAuthSession checks an AuthSession record for completeness and
// consistency before it is persisted or replayed. It validates:
//
//   - SessionID is non-empty
//   - TxHash is exactly 64 hexadecimal characters
//   - Network is a recognised Stellar network value
//   - RPCURL, when set, is a well-formed http(s) URL
//   - StartedAt is non-zero
//   - CompletedAt, when set, is not before StartedAt
//   - AuthEventCount, FailureCount and MissingSignatureCount are non-negative
//   - Status is one of the supported terminal states
//   - Error is set when (and only when) Status is "failed"
//   - FailureCount and MissingSignatureCount are consistent with the status
//
// The function never modifies the session and is safe to call concurrently.
func ValidateAuthSession(as *AuthSession) *AuthSessionReport {
	report := &AuthSessionReport{}
	if as == nil {
		report.Issues = append(report.Issues, AuthSessionIssue{
			Field:       "AuthSession",
			Description: "auth session record is nil",
			Hint:        "Build the record from the auth-debug run before validating it.",
		})
		return report
	}

	if as.SessionID == "" {
		report.Issues = append(report.Issues, AuthSessionIssue{
			Field:       "SessionID",
			Description: "auth session is missing the parent session ID",
			Hint:        "Run 'glassbox debug <tx-hash>' first to create a session, then analyze its authorization.",
		})
	}

	validateAuthTxHash(report, as.TxHash)

	if as.Network == "" {
		report.Issues = append(report.Issues, AuthSessionIssue{
			Field:       "Network",
			Description: "auth session is missing the network",
			Hint:        "Specify the network with --network testnet|mainnet|futurenet.",
		})
	} else if !validNetworks[strings.ToLower(as.Network)] {
		report.Issues = append(report.Issues, AuthSessionIssue{
			Field:       "Network",
			Description: fmt.Sprintf("auth session network %q is not a recognised Stellar network", as.Network),
			Hint:        "Accepted values are: testnet, mainnet, futurenet.",
		})
	}

	if as.RPCURL != "" {
		if err := validateAuthRPCURL(as.RPCURL); err != nil {
			report.Issues = append(report.Issues, AuthSessionIssue{
				Field:       "RPCURL",
				Description: fmt.Sprintf("auth session rpc_url %q is not valid: %v", as.RPCURL, err),
				Hint:        "Provide an http(s) URL, e.g. --rpc-url https://horizon-testnet.stellar.org.",
			})
		}
	}

	if as.StartedAt.IsZero() {
		report.Issues = append(report.Issues, AuthSessionIssue{
			Field:       "StartedAt",
			Description: "auth session has a zero started_at timestamp",
			Hint:        "This is a data-integrity issue; delete the record and re-run 'glassbox auth-debug'.",
		})
	}

	if !as.CompletedAt.IsZero() && !as.StartedAt.IsZero() && as.CompletedAt.Before(as.StartedAt) {
		report.Issues = append(report.Issues, AuthSessionIssue{
			Field:       "CompletedAt",
			Description: "completed_at is before started_at — timestamps are inconsistent",
			Hint:        "The auth session record is corrupt; delete it and re-run 'glassbox auth-debug'.",
		})
	}

	validateAuthCount(report, "AuthEventCount", as.AuthEventCount)
	validateAuthCount(report, "FailureCount", as.FailureCount)
	validateAuthCount(report, "MissingSignatureCount", as.MissingSignatureCount)

	validateAuthStatus(report, as)

	report.OK = len(report.Issues) == 0
	return report
}

// validNetworks is the shared set of recognised Stellar network names. It is
// package-level so AuthSession and ProfileSession validation agree on the set.
var validNetworks = map[string]bool{
	"testnet": true, "mainnet": true, "futurenet": true,
}

// validateAuthTxHash appends an issue when txHash is not exactly 64 hex chars.
func validateAuthTxHash(report *AuthSessionReport, txHash string) {
	if txHash == "" {
		report.Issues = append(report.Issues, AuthSessionIssue{
			Field:       "TxHash",
			Description: "auth session is missing the transaction hash",
			Hint:        "Re-run 'glassbox auth-debug <tx-hash>' with a valid transaction hash.",
		})
		return
	}
	if len(txHash) != txHashLength {
		report.Issues = append(report.Issues, AuthSessionIssue{
			Field:       "TxHash",
			Description: fmt.Sprintf("transaction hash must be exactly %d characters, got %d", txHashLength, len(txHash)),
			Hint:        "Transaction hashes are 64 hexadecimal characters; check the value you passed to auth-debug.",
		})
		return
	}
	if _, err := hex.DecodeString(txHash); err != nil {
		report.Issues = append(report.Issues, AuthSessionIssue{
			Field:       "TxHash",
			Description: fmt.Sprintf("transaction hash %q contains non-hexadecimal characters", txHash),
			Hint:        "Transaction hashes must contain only the characters 0-9 and a-f.",
		})
	}
}

// validateAuthRPCURL mirrors the http(s)+host requirement used by the auth-debug
// command's --rpc-url validation, kept local so session has no rpc dependency.
func validateAuthRPCURL(rpcURL string) error {
	u, err := url.Parse(rpcURL)
	if err != nil {
		return err
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("scheme must be http or https, got %q", u.Scheme)
	}
	if u.Host == "" {
		return fmt.Errorf("missing host")
	}
	return nil
}

// validateAuthCount appends an issue when a count field is negative.
func validateAuthCount(report *AuthSessionReport, field string, value int) {
	if value < 0 {
		report.Issues = append(report.Issues, AuthSessionIssue{
			Field:       field,
			Description: fmt.Sprintf("%s is negative (%d)", field, value),
			Hint:        "Counts cannot be negative; the record is corrupt and should be regenerated.",
		})
	}
}

// validateAuthStatus checks the Status field and its consistency with Error and
// the recorded failure counts.
func validateAuthStatus(report *AuthSessionReport, as *AuthSession) {
	if as.Status == "" {
		report.Issues = append(report.Issues, AuthSessionIssue{
			Field:       "Status",
			Description: "auth session is missing a status",
			Hint:        fmt.Sprintf("Set one of: %s.", strings.Join(SupportedAuthSessionStatuses, ", ")),
		})
		return
	}

	known := false
	for _, s := range SupportedAuthSessionStatuses {
		if as.Status == s {
			known = true
			break
		}
	}
	if !known {
		report.Issues = append(report.Issues, AuthSessionIssue{
			Field:       "Status",
			Description: fmt.Sprintf("auth session status %q is not recognised", as.Status),
			Hint:        fmt.Sprintf("Accepted values are: %s.", strings.Join(SupportedAuthSessionStatuses, ", ")),
		})
		return
	}

	switch as.Status {
	case "failed":
		if as.Error == "" {
			report.Issues = append(report.Issues, AuthSessionIssue{
				Field:       "Error",
				Description: "status is \"failed\" but no error message was recorded",
				Hint:        "Record the underlying error so the failure is actionable, or set a non-failed status.",
			})
		}
	default:
		if as.Error != "" {
			report.Issues = append(report.Issues, AuthSessionIssue{
				Field:       "Error",
				Description: fmt.Sprintf("status is %q but an error message is set (%q)", as.Status, as.Error),
				Hint:        "Clear the error message, or set the status to \"failed\".",
			})
		}
		if as.Status == "no_auth_data" && as.FailureCount > 0 {
			report.Issues = append(report.Issues, AuthSessionIssue{
				Field:       "FailureCount",
				Description: fmt.Sprintf("status is \"no_auth_data\" but %d failures are recorded", as.FailureCount),
				Hint:        "A run with authorization failures should use status \"completed\".",
			})
		}
	}
}
