// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

// Issue #1273: Ledger Entry Mocking CLI & API
package simulator

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type LedgerOverrideManifest struct {
	LedgerEntries map[string]string `json:"ledger_entries,omitempty"`
}

// LoadLedgerOverrideManifest reads a JSON manifest file and returns the
// ledger_entries map. It returns actionable errors for the common failure
// modes: file not found, unreadable, malformed JSON, and invalid base64
// entry values.
func LoadLedgerOverrideManifest(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("--mock-ledger-manifest: file not found: %q\n"+
				"  Ensure the path is correct and the file exists before running the debug command.", path)
		}
		return nil, fmt.Errorf("--mock-ledger-manifest: cannot read %q: %v", path, err)
	}

	var manifest LedgerOverrideManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("--mock-ledger-manifest: failed to parse %q as JSON: %v\n"+
			"  The file must be a JSON object with a \"ledger_entries\" key mapping base64 keys to base64 values.", path, err)
	}

	// Validate that all values are non-empty and valid base64.
	for k, v := range manifest.LedgerEntries {
		if v == "" {
			return nil, fmt.Errorf("--mock-ledger-manifest: entry %q in %q has an empty value\n"+
				"  Each ledger_entries value must be a non-empty base64-encoded XDR string.", k, path)
		}
		if _, decErr := base64.StdEncoding.DecodeString(v); decErr != nil {
			return nil, fmt.Errorf("--mock-ledger-manifest: entry %q in %q has an invalid base64 value: %v\n"+
				"  Values must be base64-encoded XDR ledger entry data.", k, path, decErr)
		}
	}

	return manifest.LedgerEntries, nil
}

// ParseLedgerOverrideFlags parses a slice of "key:value" strings supplied via
// --mock-ledger-entry flags. Each key and value must be non-empty; values
// must be valid base64.
func ParseLedgerOverrideFlags(entries []string) (map[string]string, error) {
	overrides := make(map[string]string)
	for _, entry := range entries {
		parts := strings.SplitN(entry, ":", 2)
		if len(parts) != 2 || parts[0] == "" {
			return nil, fmt.Errorf("--mock-ledger-entry: invalid format %q — expected key:value\n"+
				"  Both key and value must be non-empty base64-encoded strings.", entry)
		}
		key := parts[0]
		val := parts[1]
		if val == "" {
			return nil, fmt.Errorf("--mock-ledger-entry: entry %q has an empty value\n"+
				"  The value after the colon must be a non-empty base64-encoded XDR string.", entry)
		}
		if _, decErr := base64.StdEncoding.DecodeString(val); decErr != nil {
			return nil, fmt.Errorf("--mock-ledger-entry: entry %q has an invalid base64 value: %v\n"+
				"  Values must be base64-encoded XDR ledger entry data.", entry, decErr)
		}
		overrides[key] = val
	}

	return overrides, nil
}

func MergeLedgerOverrides(base map[string]string, overrides map[string]string) map[string]string {
	if len(overrides) == 0 {
		return base
	}

	// Always allocate a new map so the caller's base is never mutated.
	merged := make(map[string]string, len(base)+len(overrides))
	for key, value := range base {
		merged[key] = value
	}
	for key, value := range overrides {
		merged[key] = value
	}

	return merged
}
