// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

package trace

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/dotandev/glassbox/internal/errors"
)

// SaveToFile saves an ExecutionTrace to a JSON file
func (t *ExecutionTrace) SaveToFile(path string) error {
	if t == nil {
		return errors.WrapValidationError("trace is nil")
	}

	data, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		return errors.WrapMarshalFailed(err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return errors.WrapValidationError(fmt.Sprintf("failed to write trace file: %v", err))
	}

	return nil
}

// LoadExecutionTrace loads an ExecutionTrace from a JSON file.
// It understands all three on-disk shapes that Glassbox can produce:
//   - ExportJSON envelope (schema_version string key) — written by --output-json
//   - VersionedTrace envelope (version semver object) — written by --export --format json
//   - Legacy plain ExecutionTrace JSON — written by SaveToFile / older CLI versions
//
// Using LoadVersionedTrace here ensures that a file produced by any CLI export
// path can be loaded correctly, rather than the previous behaviour where only
// the plain-JSON shape was accepted and the other two would unmarshal silently
// into a structurally broken ExecutionTrace.
func LoadExecutionTrace(path string) (*ExecutionTrace, error) {
	tr, err := LoadVersionedTrace(path, DefaultCompatibilityOptions())
	if err != nil {
		return nil, errors.WrapValidationError(fmt.Sprintf(
			"failed to load trace file %q: %v\n"+
				"  The file must be a valid Glassbox JSON trace (produced by\n"+
				"  'glassbox debug --trace-output', 'glassbox trace --output-json', or\n"+
				"  'glassbox trace --export --format json')",
			path, err,
		))
	}
	return tr, nil
}
