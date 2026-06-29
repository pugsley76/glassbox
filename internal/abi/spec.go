// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

package abi

import (
	"bytes"
	"fmt"

	"github.com/stellar/go-stellar-sdk/xdr"
)

// ContractSpec holds the decoded Soroban contract specification, grouped by
// entry kind.
type ContractSpec struct {
	Functions  []xdr.ScSpecFunctionV0
	Structs    []xdr.ScSpecUdtStructV0
	Unions     []xdr.ScSpecUdtUnionV0
	Enums      []xdr.ScSpecUdtEnumV0
	ErrorEnums []xdr.ScSpecUdtErrorEnumV0
	Events     []xdr.ScSpecEventV0

	// Metadata extracted from contract manifest/ABI
	Metadata ContractMetadata
}

// ContractMetadata holds extracted metadata from Soroban contract manifests or ABI specs.
type ContractMetadata struct {
	// Name is the contract name as defined in the manifest
	Name string `json:"name,omitempty"`
	// Version is the contract version from the manifest
	Version string `json:"version,omitempty"`
	// Description is the contract description from comments/annotations
	Description string `json:"description,omitempty"`
	// Author is the contract author from metadata
	Author string `json:"author,omitempty"`
	// License is the contract license identifier
	License string `json:"license,omitempty"`
	// SourceFile is the primary source file that defines this contract
	SourceFile string `json:"source_file,omitempty"`
	// BuildInfo contains build-related metadata
	BuildInfo BuildInfo `json:"build_info,omitempty"`
}

// BuildInfo contains build information from the contract.
type BuildInfo struct {
	// RustVersion is the Rust compiler version used
	RustVersion string `json:"rust_version,omitempty"`
	// CargoVersion is the Cargo version used
	CargoVersion string `json:"cargo_version,omitempty"`
	// BuildTimestamp is when the contract was built
	BuildTimestamp string `json:"build_timestamp,omitempty"`
	// Profile is the build profile (debug, release, etc.)
	Profile string `json:"profile,omitempty"`
	// Target is the compilation target (e.g., wasm32-unknown-unknown)
	Target string `json:"target,omitempty"`
}

// HasMetadata returns true if any metadata fields are populated.
func (cm ContractMetadata) HasMetadata() bool {
	return cm.Name != "" || cm.Version != "" || cm.Description != "" ||
		cm.Author != "" || cm.License != "" || cm.SourceFile != "" ||
		cm.BuildInfo.RustVersion != "" || cm.BuildInfo.CargoVersion != "" ||
		cm.BuildInfo.BuildTimestamp != "" || cm.BuildInfo.Profile != "" || cm.BuildInfo.Target != ""
}

// String returns a human-readable representation of the metadata.
func (cm ContractMetadata) String() string {
	if !cm.HasMetadata() {
		return ""
	}
	result := ""
	if cm.Name != "" {
		result += "name: " + cm.Name + "\n"
	}
	if cm.Version != "" {
		result += "version: " + cm.Version + "\n"
	}
	if cm.Description != "" {
		result += "description: " + cm.Description + "\n"
	}
	if cm.Author != "" {
		result += "author: " + cm.Author + "\n"
	}
	if cm.License != "" {
		result += "license: " + cm.License + "\n"
	}
	if cm.SourceFile != "" {
		result += "source: " + cm.SourceFile + "\n"
	}
	if cm.BuildInfo.RustVersion != "" {
		result += "rust: " + cm.BuildInfo.RustVersion + "\n"
	}
	if cm.BuildInfo.Profile != "" {
		result += "profile: " + cm.BuildInfo.Profile + "\n"
	}
	return result
}

// DecodeContractSpec reads concatenated XDR-encoded ScSpecEntry values and
// returns them grouped by kind.
func DecodeContractSpec(data []byte) (*ContractSpec, error) {
	spec := &ContractSpec{}
	reader := bytes.NewReader(data)

	for reader.Len() > 0 {
		var entry xdr.ScSpecEntry
		_, err := xdr.Unmarshal(reader, &entry)
		if err != nil {
			return nil, fmt.Errorf("decoding spec entry: %w", err)
		}

		switch entry.Kind {
		case xdr.ScSpecEntryKindScSpecEntryFunctionV0:
			spec.Functions = append(spec.Functions, *entry.FunctionV0)
		case xdr.ScSpecEntryKindScSpecEntryUdtStructV0:
			spec.Structs = append(spec.Structs, *entry.UdtStructV0)
		case xdr.ScSpecEntryKindScSpecEntryUdtUnionV0:
			spec.Unions = append(spec.Unions, *entry.UdtUnionV0)
		case xdr.ScSpecEntryKindScSpecEntryUdtEnumV0:
			spec.Enums = append(spec.Enums, *entry.UdtEnumV0)
		case xdr.ScSpecEntryKindScSpecEntryUdtErrorEnumV0:
			spec.ErrorEnums = append(spec.ErrorEnums, *entry.UdtErrorEnumV0)
		case xdr.ScSpecEntryKindScSpecEntryEventV0:
			spec.Events = append(spec.Events, *entry.EventV0)
		default:
			return nil, fmt.Errorf("unknown spec entry kind: %d", entry.Kind)
		}
	}

	return spec, nil
}
