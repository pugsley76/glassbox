// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"

	"github.com/dotandev/glassbox/internal/errors"
	"github.com/dotandev/glassbox/internal/termctx"
	"github.com/spf13/cobra"
)

// requireInteractiveTTY returns an error when the command is running in
// non-interactive mode (CI, piped stdin, --non-interactive flag). Commands
// that need user prompts MUST call this before reading from stdin.
func requireInteractiveTTY(cmd *cobra.Command, action string) error {
	if termctx.GlobalNonInteractive() {
		return errors.WrapValidationError(fmt.Sprintf(
			"non-interactive mode: %s requires an interactive terminal; use --yes or --force to skip prompting",
			action,
		))
	}
	return nil
}

// confirmWithForceOrNonInteractive handles the common pattern of prompting for
// confirmation with a force/skip flag. When forceFlag is true, returns defaultVal
// immediately. When non-interactive, returns an actionable error. Otherwise,
// prints the prompt and reads the user's answer.
func confirmWithForceOrNonInteractive(
	cmd *cobra.Command,
	prompt string,
	forceFlag bool,
	defaultVal bool,
) (bool, error) {
	if forceFlag {
		return defaultVal, nil
	}
	if termctx.GlobalNonInteractive() {
		return defaultVal, errors.WrapValidationError(fmt.Sprintf(
			"non-interactive mode: confirmation prompt requires an interactive terminal; "+
				"use --yes or --force to skip",
		))
	}
	return promptYesNo(cmd, prompt), nil
}

// requireInteractiveShell returns an error when the shell command is running
// in non-interactive mode. The shell command is inherently interactive and
// cannot function without a terminal.
func requireInteractiveShell(cmd *cobra.Command) error {
	if termctx.GlobalNonInteractive() {
		return errors.WrapValidationError(
			"non-interactive mode: the shell command requires an interactive terminal; " +
				"use 'glassbox debug' or individual commands instead",
		)
	}
	return nil
}

// CacheCleanupAutomationExample is the documented CI recipe for bounded cache
// cleanup. It is kept next to the non-interactive helpers so the flags and the
// force requirements stay in sync with the code that enforces them.
//
// Impact: `cache rpc --older-than 7 --network testnet` only removes cached RPC
// responses for the testnet network that are older than 7 days. It never
// touches other networks, other cache kinds, or entries newer than the cutoff.
//
// Non-interactive rules:
//   - `cache clean` and `cache clear` prompt for confirmation and therefore
//     require `--force` when termctx.GlobalNonInteractive() is true (see
//     internal/cache/cache.go and confirmWithForceOrNonInteractive above).
//   - A full `cache clear` is not routine CI behavior: it discards every cache
//     entry and forces cold rebuilds. Prefer age/network-scoped cleanup.
const CacheCleanupAutomationExample = `# Bounded cache cleanup for CI.
#
# Impact: removes only cached RPC responses for the testnet network that are
# older than 7 days. Other networks, other cache kinds, and newer entries are
# left untouched. This is NOT a full cache clear.
#
# Note: 'cache clean' and 'cache clear' prompt for confirmation and require
# --force in non-interactive mode. Avoid a blanket 'cache clear' in CI; it
# discards all entries and forces cold rebuilds.
set -euo pipefail

if ! glassbox cache rpc --older-than 7 --network testnet; then
  echo "cache rpc cleanup failed" >&2
  exit 1
fi
`
