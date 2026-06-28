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
	b, _ := json.Marshal(info)
	sum := sha256.Sum256(b)
	s := hex.EncodeToString(sum[:])
	return fmt.Sprintf("sha256:%s", s[:32])
}
