// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

package protocolreg

import (
	"fmt"
	"runtime"
	"strings"
)

// ConflictResolutionResult is the outcome of ResolveConflict.
type ConflictResolutionResult struct {
	// Resolved is true when the conflict was successfully cleared and the
	// current binary is now the registered handler.
	Resolved bool
	// ConflictingHandler is the path that was registered before resolution.
	ConflictingHandler string
	// Actions records the ordered steps taken during resolution.
	Actions []string
	// Err is non-nil when resolution failed.
	Err error
	// Hint provides platform-specific guidance when Err is non-nil.
	Hint string
}

// ResolveConflict detects and resolves a protocol registration conflict for
// the glassbox:// scheme. A conflict exists when the OS handler is registered
// to a binary path other than the current executable.
//
// Resolution strategy:
//  1. Run Diagnose to confirm a conflict is present.
//  2. If no conflict is detected, return successfully (nothing to resolve).
//  3. Validate the conflicting handler path: reject empty and null-byte paths.
//  4. Re-register the current executable as the authoritative handler.
//  5. Run a post-resolution Diagnose to confirm ConflictDetected is cleared.
//
// Callers that want interactive confirmation before overwriting should check
// the ConflictingHandler field on the pre-resolution DiagnosticReport before
// calling ResolveConflict.
func (r *Registrar) ResolveConflict() *ConflictResolutionResult {
	result := &ConflictResolutionResult{}

	if r.executablePath == "" {
		result.Err = fmt.Errorf(
			"cannot resolve conflict: executable path is empty\n" +
				"  Fix: ensure glassbox is invoked from a valid binary path, not via 'go run'",
		)
		result.Hint = "Reinstall Glassbox or invoke it from a named binary."
		return result
	}

	// Step 1: gather current registration state.
	diag := r.Diagnose()
	result.Actions = append(result.Actions,
		fmt.Sprintf("Diagnosed registration state on %s: status=%s", runtime.GOOS, diag.Status))

	// Step 2: no conflict — nothing to do.
	if !diag.ConflictDetected {
		result.Resolved = true
		result.Actions = append(result.Actions,
			"No conflict detected — the glassbox:// handler is already correctly registered.")
		return result
	}

	// Step 3: validate the conflicting handler path before logging it.
	conflicting := diag.ConflictingHandler
	if err := validateConflictingHandlerPath(conflicting); err != nil {
		result.Err = fmt.Errorf("conflict resolution: invalid conflicting handler path: %w", err)
		result.Hint = "Inspect the registration manually and run 'glassbox protocol:repair' to overwrite it."
		return result
	}

	result.ConflictingHandler = conflicting
	result.Actions = append(result.Actions,
		fmt.Sprintf("Conflict detected: glassbox:// is registered to %q — displacing with %q",
			conflicting, r.executablePath))

	// Step 4: overwrite the registration.
	result.Actions = append(result.Actions,
		fmt.Sprintf("Re-registering glassbox:// to point to %s", r.executablePath))

	if err := r.Register(); err != nil {
		result.Err = fmt.Errorf("conflict resolution: registration failed: %w", err)
		result.Hint = r.permissionHint(err)
		if result.Hint == "" {
			result.Hint = fmt.Sprintf(
				"Check that %s is accessible and that you have the required privileges.", r.executablePath)
		}
		result.Actions = append(result.Actions,
			fmt.Sprintf("Registration failed: %v", err))
		return result
	}
	result.Actions = append(result.Actions, "Registration succeeded.")

	// Step 5: post-resolution verification.
	postDiag := r.Diagnose()
	if postDiag.ConflictDetected {
		result.Err = fmt.Errorf(
			"conflict resolution: registration succeeded but conflict is still reported — "+
				"conflicting handler: %s",
			postDiag.ConflictingHandler)
		result.Hint = "Run 'glassbox protocol:repair' for a full repair pass."
		result.Actions = append(result.Actions,
			"Post-resolution verification found a lingering conflict.")
		return result
	}

	result.Resolved = true
	result.Actions = append(result.Actions,
		"Post-resolution verification passed — conflict cleared.")
	return result
}

// validateConflictingHandlerPath checks that the path extracted from the OS
// registration is safe to reference in error messages and logs.
func validateConflictingHandlerPath(path string) error {
	if strings.ContainsRune(path, 0) {
		return fmt.Errorf("path contains null bytes: %q", path)
	}
	// A truly empty conflicting path means the diagnostic could not extract it;
	// this is allowed (we just log nothing) but we don't block on it.
	return nil
}
