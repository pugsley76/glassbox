// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

package plugin

import (
	"encoding/json"
	"fmt"
	"os"
)

// Policy defines the sandbox policy for plugin loading.
// It controls which capabilities and permissions are denied globally,
// regardless of what individual plugin manifests declare.
type Policy struct {
	// DeniedCapabilities is the set of capability strings that are forbidden.
	// Any plugin declaring one of these capabilities will be refused at load time.
	DeniedCapabilities []string `json:"denied_capabilities,omitempty"`

	// DeniedPermissions is the set of permission strings that are forbidden.
	// Any plugin requesting one of these permissions will be refused at load time.
	DeniedPermissions []string `json:"denied_permissions,omitempty"`

	// DeniedPlugins is a list of plugin names that are explicitly blocked,
	// regardless of their capabilities or permissions.
	DeniedPlugins []string `json:"denied_plugins,omitempty"`

	// AllowUntrusted, when true, permits plugins with TrustLevel 'untrusted'
	// to load. The zero value is false (deny untrusted); use DefaultPolicy()
	// to get a permissive policy.
	AllowUntrusted bool `json:"allow_untrusted"`
}

// DefaultPolicy returns a permissive policy that denies nothing.
// This preserves backward-compatible behaviour when no policy file is present.
func DefaultPolicy() *Policy {
	return &Policy{AllowUntrusted: true}
}

// LoadPolicy reads and parses a JSON policy file from path.
func LoadPolicy(path string) (*Policy, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read policy file %q: %w", path, err)
	}
	var p Policy
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("failed to parse policy file %q: %w", path, err)
	}
	return &p, nil
}

// CheckManifest returns an error if the manifest violates the policy.
// It checks denied capabilities, denied permissions, denied plugin names,
// and the untrusted-plugin allowance.
func (p *Policy) CheckManifest(m *Manifest) error {
	if p == nil {
		return nil
	}

	// Check denied plugin names.
	for _, denied := range p.DeniedPlugins {
		if m.Name == denied {
			return fmt.Errorf("plugin %q is explicitly denied by policy", m.Name)
		}
	}

	// Check denied capabilities.
	for _, deniedCap := range p.DeniedCapabilities {
		for _, cap := range m.Capabilities {
			if string(cap) == deniedCap {
				return fmt.Errorf("plugin %q declares capability %q which is denied by policy", m.Name, deniedCap)
			}
		}
	}

	// Check denied permissions.
	for _, deniedPerm := range p.DeniedPermissions {
		for _, perm := range m.Permissions {
			if string(perm) == deniedPerm {
				return fmt.Errorf("plugin %q requests permission %q which is denied by policy", m.Name, deniedPerm)
			}
		}
	}

	// Check untrusted allowance.
	if !p.AllowUntrusted && (m.TrustLevel == TrustLevelUntrusted || m.TrustLevel == "") {
		return fmt.Errorf("plugin %q has trust level %q which is not allowed by policy", m.Name, m.TrustLevel)
	}

	return nil
}
