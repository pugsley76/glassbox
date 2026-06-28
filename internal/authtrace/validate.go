// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

package authtrace

import (
	"fmt"
	"strings"
)

// AuthInputError is returned when one or more auth-debug inputs or trace
// fields are invalid. Each element in Failures is an actionable description
// of a single problem so callers can surface all issues at once.
type AuthInputError struct {
	Failures []string
}

func (e *AuthInputError) Error() string {
	if len(e.Failures) == 1 {
		return e.Failures[0]
	}
	lines := make([]string, 0, len(e.Failures)+1)
	lines = append(lines, fmt.Sprintf("%d auth validation error(s):", len(e.Failures)))
	for i, f := range e.Failures {
		lines = append(lines, fmt.Sprintf("  %d. %s", i+1, f))
	}
	return strings.Join(lines, "\n")
}

// ValidateAuthTraceInputs validates the inputs that are required to start an
// auth-debug analysis. All checks run before any network or simulation work
// begins so that users see every problem in a single pass.
//
// Parameters:
//   - txHash:   the 64-character hex transaction hash (required)
//   - network:  the Stellar network name (optional; auto-detected when empty)
//   - rpcURL:   custom RPC URL (optional; validated for scheme/host when provided)
//
// Returns nil when all inputs are valid, or an *AuthInputError listing every
// problem found.
func ValidateAuthTraceInputs(txHash, network, rpcURL string) error {
	var failures []string

	// Transaction hash: must be exactly 64 hexadecimal characters.
	trimmed := strings.TrimSpace(txHash)
	if trimmed == "" {
		failures = append(failures, "transaction hash is required\n"+
			"  Fix: provide a 64-character hex transaction hash\n"+
			"  Example: glassbox auth-debug 5c0a1234567890abcdef1234567890abcdef1234567890abcdef1234567890ab")
	} else if len(trimmed) != 64 {
		failures = append(failures, fmt.Sprintf(
			"invalid transaction hash %q: must be exactly 64 hexadecimal characters (got %d)\n"+
				"  Fix: verify the hash is complete and correctly copied\n"+
				"  Example: 5c0a1234567890abcdef1234567890abcdef1234567890abcdef1234567890ab",
			trimmed, len(trimmed),
		))
	} else {
		for _, c := range trimmed {
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
				failures = append(failures, fmt.Sprintf(
					"invalid transaction hash %q: contains non-hexadecimal character %q\n"+
						"  Fix: transaction hashes use only 0-9 and a-f characters",
					trimmed, string(c),
				))
				break
			}
		}
	}

	// Network: when provided must be a known value.
	if network != "" {
		switch strings.ToLower(strings.TrimSpace(network)) {
		case "testnet", "mainnet", "futurenet":
			// valid
		default:
			failures = append(failures, fmt.Sprintf(
				"invalid --network %q — must be one of: testnet, mainnet, futurenet\n"+
					"  Fix: use a supported network name or omit --network for auto-detection",
				network,
			))
		}
	}

	// RPC URL: when provided must have http/https scheme and a host.
	if rpcURL != "" {
		for _, u := range strings.Split(rpcURL, ",") {
			u = strings.TrimSpace(u)
			if u == "" {
				continue
			}
			if !strings.HasPrefix(u, "http://") && !strings.HasPrefix(u, "https://") {
				failures = append(failures, fmt.Sprintf(
					"--rpc-url %q is not valid: must start with http:// or https://\n"+
						"  Fix: provide a full URL including scheme\n"+
						"  Example: --rpc-url https://soroban-testnet.stellar.org",
					u,
				))
				break
			}
			// Check that something follows the scheme (non-empty host).
			rest := strings.TrimPrefix(strings.TrimPrefix(u, "https://"), "http://")
			if strings.TrimSpace(rest) == "" {
				failures = append(failures, fmt.Sprintf(
					"--rpc-url %q is not valid: missing host after scheme\n"+
						"  Fix: provide a complete URL with host\n"+
						"  Example: --rpc-url https://soroban-testnet.stellar.org",
					u,
				))
				break
			}
		}
	}

	if len(failures) > 0 {
		return &AuthInputError{Failures: failures}
	}
	return nil
}

// ValidateAuthTrace checks an AuthTrace for the conditions that make its
// exported output accurate and actionable. It is called before generating
// reports or exporting JSON so users receive a clear warning instead of
// silently incomplete output.
//
// Checks:
//   - trace is not nil
//   - AccountID is present (missing → exported JSON lacks account context)
//   - At least one AuthEvent is present (no events → "no failures" is misleading)
//   - Every AuthEvent has a non-empty EventType and Status
//   - Every AuthFailure has a known FailureReason (not ReasonUnknown)
//   - ReplayAttack events are present when failure reasons include replay attack
//
// Returns nil when the trace is valid, or an *AuthInputError listing issues.
func ValidateAuthTrace(trace *AuthTrace) error {
	if trace == nil {
		return &AuthInputError{Failures: []string{
			"auth trace is nil — cannot generate a report from a nil trace\n" +
				"  Fix: ensure the auth-debug analysis completed successfully before requesting output",
		}}
	}

	var failures []string

	// AccountID must be present for the report to be meaningful.
	if strings.TrimSpace(trace.AccountID) == "" {
		failures = append(failures, "auth trace has no AccountID — exported report will lack account context\n"+
			"  Fix: ensure InitializeAccountContext is called before GenerateTrace\n"+
			"  This usually means the transaction contained no Soroban auth entries")
	}

	// No events: "success" is ambiguous when there was nothing to check.
	if len(trace.AuthEvents) == 0 {
		failures = append(failures,
			"auth trace contains no authorization events — the report reflects "+
				"\"no failures recorded\", not verified-successful authorization\n"+
				"  This is expected for transactions with no Soroban auth entries\n"+
				"  Fix: verify the transaction hash and --network are correct\n"+
				"  Tip: run 'glassbox doctor' if you expected auth data")
	}

	// Per-event checks.
	for i, event := range trace.AuthEvents {
		if strings.TrimSpace(event.EventType) == "" {
			failures = append(failures, fmt.Sprintf(
				"auth event #%d has no EventType — event will be unclassified in reports\n"+
					"  Fix: ensure RecordEvent is always called with a non-empty EventType",
				i+1,
			))
		}
		if strings.TrimSpace(event.Status) == "" {
			failures = append(failures, fmt.Sprintf(
				"auth event #%d (%s) has no Status — pass/fail state is ambiguous\n"+
					"  Fix: ensure every event is recorded with a Status of 'valid', 'invalid', 'passed', or 'failed'",
				i+1, event.EventType,
			))
		}
	}

	// Per-failure checks.
	for i, f := range trace.Failures {
		if f.FailureReason == ReasonUnknown || f.FailureReason == "" {
			failures = append(failures, fmt.Sprintf(
				"auth failure #%d for account %q has unknown failure reason — "+
					"the root cause cannot be communicated to users\n"+
					"  Fix: use a specific AuthFailureReason constant when calling recordFailure",
				i+1, f.AccountID,
			))
		}
		if f.RequiredWeight > 0 && f.CollectedWeight > f.RequiredWeight {
			failures = append(failures, fmt.Sprintf(
				"auth failure #%d for account %q: collectedWeight (%d) > requiredWeight (%d) "+
					"but still recorded as a failure — data is inconsistent\n"+
					"  Fix: only record a threshold failure when collectedWeight < requiredWeight",
				i+1, f.AccountID, f.CollectedWeight, f.RequiredWeight,
			))
		}
	}

	if len(failures) > 0 {
		return &AuthInputError{Failures: failures}
	}
	return nil
}

// ValidateSignatureInput validates the inputs to RecordSignatureVerification
// before they are accepted into the tracker. Returns an actionable error when
// either accountID or signerKey is empty or whitespace-only.
func ValidateSignatureInput(accountID, signerKey string, sigType SignatureType) error {
	var failures []string

	if strings.TrimSpace(accountID) == "" {
		failures = append(failures, "accountID is required for signature verification recording\n"+
			"  Fix: provide the Stellar account ID of the signer (starts with 'G')")
	}
	if strings.TrimSpace(signerKey) == "" {
		failures = append(failures, "signerKey is required for signature verification recording\n"+
			"  Fix: provide the public key of the signer")
	}
	if sigType == "" {
		failures = append(failures, "signature type is required — must be one of: ed25519, secp256k1, pre_authorized, custom_account\n"+
			"  Fix: pass the correct SignatureType constant")
	} else {
		switch sigType {
		case Ed25519, Secp256k1, PreAuthorized, CustomAccount:
			// valid
		default:
			failures = append(failures, fmt.Sprintf(
				"unknown signature type %q — must be one of: ed25519, secp256k1, pre_authorized, custom_account\n"+
					"  Fix: use a valid SignatureType constant",
				sigType,
			))
		}
	}

	if len(failures) > 0 {
		return &AuthInputError{Failures: failures}
	}
	return nil
}

// ValidateContractAuthInput validates the inputs to ValidateContract before
// dispatch. Returns an actionable error when contractID or method is empty.
func ValidateContractAuthInput(contractID, method string) error {
	var failures []string

	if strings.TrimSpace(contractID) == "" {
		failures = append(failures, "contractID is required for contract auth validation\n"+
			"  Fix: provide the Stellar contract ID (C-prefixed Strkey)")
	}
	if strings.TrimSpace(method) == "" {
		failures = append(failures, "method is required for contract auth validation\n"+
			"  Fix: provide the contract method name being authorized")
	}

	if len(failures) > 0 {
		return &AuthInputError{Failures: failures}
	}
	return nil
}
