// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

package session

import (
	"fmt"
	"os"
	"regexp"
)

// piiPatterns is a list of regex patterns that match potentially sensitive
// information that must never be surfaced in error messages or logs.
var piiPatterns = []*regexp.Regexp{
	// Home-directory path prefixes (Unix and Windows)
	regexp.MustCompile(`(?i)(/home/[^/\s]+|/Users/[^/\s]+|C:\\Users\\[^\\\s]+)`),
	// Stellar secret seeds (56-char base32 starting with 'S')
	regexp.MustCompile(`\bS[A-Z2-7]{55}\b`),
	// JWT tokens: three base64url segments separated by dots
	regexp.MustCompile(`eyJ[A-Za-z0-9_\-]{20,}\.[A-Za-z0-9_\-]{20,}\.[A-Za-z0-9_\-]{20,}`),
}

// SanitizeErrorMessage removes PII patterns from an error string before it is
// shown to the user or written to a log. Redacted sections are replaced with
// the placeholder "[REDACTED]" so the position in the message is preserved.
func SanitizeErrorMessage(msg string) string {
	for _, pat := range piiPatterns {
		msg = pat.ReplaceAllString(msg, "[REDACTED]")
	}
	return msg
}

// SanitizeDBPath replaces the user-specific home-directory portion of a
// database path with "~" so error messages never leak usernames.
func SanitizeDBPath(path string) string {
	re := regexp.MustCompile(`(?i)(/home/[^/]+|/Users/[^/]+|C:\\Users\\[^\\]+)`)
	return re.ReplaceAllString(path, "~")
}

// ValidateDBPermissions checks that the SQLite database file at dbPath is
// readable and writable by the current process. Returns a descriptive,
// PII-free error when the check fails.
func ValidateDBPermissions(dbPath string) error {
	safePath := SanitizeDBPath(dbPath)

	f, err := os.OpenFile(dbPath, os.O_RDWR, 0)
	if err != nil {
		if os.IsPermission(err) {
			return fmt.Errorf(
				"session database %q is not readable/writable (permission denied)\n"+
					"Fix: run 'chmod 600 %s' or delete and re-create the file",
				safePath, safePath,
			)
		}
		if os.IsNotExist(err) {
			// Not yet created — not an error; NewStore will create it.
			return nil
		}
		return fmt.Errorf("cannot open session database %q: %s",
			safePath, SanitizeErrorMessage(err.Error()))
	}
	_ = f.Close()
	return nil
}

// WrapStoreError wraps a store-level error with a sanitized message so neither
// the database path nor raw OS details leak user-specific information.
func WrapStoreError(operation, dbPath string, err error) error {
	if err == nil {
		return nil
	}
	safePath := SanitizeDBPath(dbPath)
	safeMsg := SanitizeErrorMessage(err.Error())
	return fmt.Errorf("session store %s failed (db: %s): %s", operation, safePath, safeMsg)
}

// RedactTxHash shortens a full transaction hash to its first 8 characters for
// display in logs, reducing on-chain traceability in plaintext output.
func RedactTxHash(txHash string) string {
	if len(txHash) <= 8 {
		return txHash
	}
	return txHash[:8] + "..."
}
