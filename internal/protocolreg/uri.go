// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

package protocolreg

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/dotandev/glassbox/internal/simulator"
)

var txHashPattern = regexp.MustCompile(`^[a-fA-F0-9]{64}$`)

const (
	maxSourceLen    = 256
	maxSignatureLen = 512
)

// allowedNetworks is the set of valid network identifiers for the deep link.
var allowedNetworks = map[string]bool{
	"testnet":   true,
	"mainnet":   true,
	"futurenet": true,
}

// allowedViews is the set of valid view mode identifiers for the deep link.
var allowedViews = map[string]bool{
	"trace":      true,
	"flamegraph": true,
	"events":     true,
	"auth":       true,
	"budget":     true,
	"storage":    true,
}

// ParsedDebugURI holds the validated fields extracted from a glassbox:// debug URI.
//
// Supported URI format:
//
//	glassbox://debug/<txhash>?network=<n>[&op=<i>][&operation=<i>][&view=<v>][&source=<s>][&signature=<s>]
//
// Query parameters:
//   - network  (required) — one of: testnet, mainnet, futurenet
//   - op       (optional) — zero-based operation index (alias for "operation")
//   - operation (optional) — zero-based operation index (legacy; "op" takes precedence)
//   - view     (optional) — initial view mode: trace, flamegraph, events, auth, budget, storage
//   - source   (optional) — free-form source identifier (e.g. "dashboard")
//   - signature (optional) — free-form signature hint
type ParsedDebugURI struct {
	// Raw is the original unmodified URI string.
	Raw string
	// TransactionHash is the 64-character lowercase hex transaction hash.
	TransactionHash string
	// Network is the validated network identifier (testnet, mainnet, futurenet).
	Network string
	// Op is the zero-based operation index, populated from the "op" or "operation" query parameter.
	// nil means no operation was specified.
	Op *int
	// Operation is an alias for Op retained for backward compatibility.
	// It always mirrors Op.
	Operation *int
	// View is the requested initial view mode (trace, flamegraph, events, auth, budget, storage).
	// Empty string means no view was specified and the default view should be used.
	View string
	// Source is an optional free-form source identifier.
	Source string
	// Signature is an optional free-form signature hint.
	Signature string
	// ProtocolVersion is the optional protocol version override for simulation.
	ProtocolVersion *uint32
	// MockLedgerManifest is the optional path to a mock ledger JSON manifest.
	MockLedgerManifest string
	// MockLedgerEntries is the optional list of mock ledger key:value overrides.
	MockLedgerEntries []string
}

// ParseDebugURI parses and validates a glassbox:// debug URI.
//
// Returns a descriptive error for each class of invalid input:
//   - empty URI
//   - null bytes or control characters
//   - wrong scheme
//   - wrong host (not "debug")
//   - missing or malformed transaction hash
//   - missing or invalid network
//   - invalid op/operation index (non-numeric or negative)
//   - unrecognised view mode
func ParseDebugURI(raw string) (*ParsedDebugURI, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("protocol URI must not be empty")
	}
	// Reject null bytes and ASCII control characters to prevent injection attacks.
	for i := 0; i < len(raw); i++ {
		if raw[i] == 0x00 {
			return nil, fmt.Errorf("protocol URI must not contain null bytes")
		}
		if raw[i] < 0x20 && raw[i] != '\t' {
			return nil, fmt.Errorf("protocol URI must not contain control characters (found 0x%02x)", raw[i])
		}
	}
	if !strings.HasPrefix(raw, Scheme+"://") {
		return nil, fmt.Errorf("invalid protocol URI: expected %s://", Scheme)
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("parse protocol URI: %w", err)
	}

	if parsed.Host != "debug" {
		return nil, fmt.Errorf("invalid protocol host %q: expected \"debug\"", parsed.Host)
	}

	transactionHash := strings.TrimPrefix(parsed.EscapedPath(), "/")
	transactionHash, err = url.PathUnescape(transactionHash)
	if err != nil {
		return nil, fmt.Errorf("decode transaction hash: %w", err)
	}
	if !txHashPattern.MatchString(transactionHash) {
		return nil, fmt.Errorf("invalid transaction hash %q: must be a 64-character hex string", transactionHash)
	}

	q := parsed.Query()

	// --- network (required) ---
	network := q.Get("network")
	if network == "" {
		return nil, fmt.Errorf("missing required query parameter: network")
	}
	if !allowedNetworks[network] {
		return nil, fmt.Errorf("invalid network %q: must be one of testnet, mainnet, futurenet", network)
	}

	source := q.Get("source")
	if len(source) > maxSourceLen {
		return nil, fmt.Errorf(
			"source parameter is too long (%d characters, max %d)",
			len(source), maxSourceLen,
		)
	}
	if strings.ContainsRune(source, 0) {
		return nil, fmt.Errorf("source parameter contains null bytes and cannot be used")
	}

	signature := q.Get("signature")
	if len(signature) > maxSignatureLen {
		return nil, fmt.Errorf(
			"signature parameter is too long (%d characters, max %d)",
			len(signature), maxSignatureLen,
		)
	}
	if strings.ContainsRune(signature, 0) {
		return nil, fmt.Errorf("signature parameter contains null bytes and cannot be used")
	}

	result := &ParsedDebugURI{
		Raw:             raw,
		TransactionHash: transactionHash,
		Network:         network,
		Source:          source,
		Signature:       signature,
	}

	// --- protocol-version (optional) ---
	protoVerStr := q.Get("protocol-version")
	if protoVerStr != "" {
		protoVer, err := strconv.ParseUint(protoVerStr, 10, 32)
		if err != nil || protoVer == 0 {
			return nil, fmt.Errorf("invalid protocol-version %q: must be a positive integer\n"+
				"  Fix: use a supported version number (e.g. 20, 21, or 22)", protoVerStr)
		}
		val := uint32(protoVer)
		if err := simulator.Validate(val); err != nil {
			return nil, fmt.Errorf("invalid protocol-version %d: %w\n"+
				"  Fix: use a supported protocol version (e.g. 20, 21, or 22)\n"+
				"  Tip: run 'glassbox version' to see all supported versions", val, err)
		}
		result.ProtocolVersion = &val
	}

	// --- mock-ledger-manifest (optional) ---
	mockManifest := q.Get("mock-ledger-manifest")
	if mockManifest != "" {
		if strings.ContainsRune(mockManifest, 0) {
			return nil, fmt.Errorf("mock-ledger-manifest parameter contains null bytes and cannot be used")
		}
		result.MockLedgerManifest = mockManifest
	}

	// --- mock-ledger-entry (optional, repeatable) ---
	mockEntries := q["mock-ledger-entry"]
	if len(mockEntries) > 0 {
		for _, entry := range mockEntries {
			entry = strings.TrimSpace(entry)
			if entry == "" {
				continue
			}
			parts := strings.SplitN(entry, ":", 2)
			if len(parts) != 2 || parts[0] == "" {
				return nil, fmt.Errorf("invalid mock-ledger-entry format %q — expected key:value\n"+
					"  Fix: specify both key and value as non-empty colon-separated strings", entry)
			}
			val := parts[1]
			if val == "" {
				return nil, fmt.Errorf("mock-ledger-entry %q has an empty value\n"+
					"  Fix: specify a non-empty base64-encoded value after the colon", entry)
			}
			if _, decErr := base64.StdEncoding.DecodeString(val); decErr != nil {
				return nil, fmt.Errorf("mock-ledger-entry %q has an invalid base64 value: %v\n"+
					"  Fix: ensure the value after the colon is valid base64", entry, decErr)
			}
			result.MockLedgerEntries = append(result.MockLedgerEntries, entry)
		}
	}

	// --- op / operation (optional, "op" takes precedence) ---
	opStr := q.Get("op")
	if opStr == "" {
		opStr = q.Get("operation")
	}
	if opStr != "" {
		parsedOp, parseErr := strconv.Atoi(opStr)
		if parseErr != nil || parsedOp < 0 {
			return nil, fmt.Errorf(
				"invalid operation index %q: must be a non-negative integer\n"+
					"  Fix: use a whole number >= 0 (e.g. op=0 for the first operation)",
				opStr,
			)
		}
		// Guard against values that parsed as int on 64-bit but would overflow
		// on 32-bit platforms or downstream consumers expecting a reasonable index.
		const maxOpIndex = 65535
		if parsedOp > maxOpIndex {
			return nil, fmt.Errorf(
				"operation index %d exceeds the maximum allowed value (%d)\n"+
					"  Fix: use an index in the range 0–%d",
				parsedOp, maxOpIndex, maxOpIndex,
			)
		}
		result.Op = &parsedOp
		result.Operation = &parsedOp
	}

	// --- view (optional) ---
	if view := q.Get("view"); view != "" {
		if !allowedViews[view] {
			return nil, fmt.Errorf("invalid view %q: must be one of trace, flamegraph, events, auth, budget, storage", view)
		}
		result.View = view
	}

	return result, nil
}
