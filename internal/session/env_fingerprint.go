// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

package session

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"runtime"

	"github.com/dotandev/glassbox/internal/version"
)

// BuildEnvFingerprint returns a short deterministic fingerprint describing
// the runtime environment (glassbox version, OS, arch, go version).
func BuildEnvFingerprint() string {
	info := map[string]string{
		"glassbox_version": version.Version,
		"goos":             runtime.GOOS,
		"goarch":           runtime.GOARCH,
		"go_version":       runtime.Version(),
	}
	b, err := json.Marshal(info)
	if err != nil {
		// Fall back to a deterministic fingerprint built from individual fields
		// without JSON encoding so that a marshal failure never produces an
		// empty or zero-value fingerprint.
		raw := version.Version + "|" + runtime.GOOS + "|" + runtime.GOARCH + "|" + runtime.Version()
		sum := sha256.Sum256([]byte(raw))
		return fmt.Sprintf("sha256:%s", hex.EncodeToString(sum[:])[:32])
	}
	sum := sha256.Sum256(b)
	return fmt.Sprintf("sha256:%s", hex.EncodeToString(sum[:])[:32])
}
