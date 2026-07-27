// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

// importer.go provides importers for external contract ABI/spec definitions.
// Supported formats:
//   - JSON ABI (glassbox canonical JSON format produced by FormatJSON)
//   - XDR spec bytes (raw concatenated ScSpecEntry XDR, same as contractspecv0 WASM section)

package abi

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/stellar/go-stellar-sdk/xdr"
)

// ImportFormat identifies the format of an external ABI/spec file.
type ImportFormat string

const (
	// ImportFormatJSON is the glassbox canonical JSON ABI format.
	ImportFormatJSON ImportFormat = "json"
	// ImportFormatXDR is raw XDR-encoded ScSpecEntry bytes (binary or base64).
	ImportFormatXDR ImportFormat = "xdr"
)

// ImportFromJSON parses a JSON ABI file (as produced by `glassbox abi --json`)
// and returns a ContractSpec. The JSON schema matches the jsonSpec struct used
// by FormatJSON in printer.go.
func ImportFromJSON(data []byte) (*ContractSpec, error) {
	var js jsonSpec
	if err := json.Unmarshal(data, &js); err != nil {
		return nil, fmt.Errorf("parsing JSON ABI: %w", err)
	}
	return jsonSpecToContractSpec(&js)
}

// ImportFromXDR parses raw XDR-encoded ScSpecEntry bytes (the same binary
// format stored in the WASM contractspecv0 custom section) and returns a
// ContractSpec. This is identical to DecodeContractSpec but exposed under a
// distinct name to make the import path explicit.
func ImportFromXDR(data []byte) (*ContractSpec, error) {
	spec, err := DecodeContractSpec(data)
	if err != nil {
		return nil, fmt.Errorf("parsing XDR spec: %w", err)
	}
	return spec, nil
}

// DetectFormat attempts to auto-detect the format of the provided data.
// It returns ImportFormatJSON when the data starts with '{', otherwise
// ImportFormatXDR.
func DetectFormat(data []byte) ImportFormat {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) > 0 && trimmed[0] == '{' {
		return ImportFormatJSON
	}
	return ImportFormatXDR
}

// jsonSpecToContractSpec converts the JSON-friendly jsonSpec representation
// back into a ContractSpec with XDR types.
func jsonSpecToContractSpec(js *jsonSpec) (*ContractSpec, error) {
	spec := &ContractSpec{}

	for _, jf := range js.Functions {
		fn := xdr.ScSpecFunctionV0{
			Name: xdr.ScSymbol(jf.Name),
			Doc:  jf.Doc,
		}
		for _, inp := range jf.Inputs {
			td, err := parseTypeDef(inp.Type)
			if err != nil {
				return nil, fmt.Errorf("function %q input %q: %w", jf.Name, inp.Name, err)
			}
			fn.Inputs = append(fn.Inputs, xdr.ScSpecFunctionInputV0{
				Name: inp.Name,
				Type: td,
			})
		}
		for _, out := range jf.Outputs {
			td, err := parseTypeDef(out)
			if err != nil {
				return nil, fmt.Errorf("function %q output %q: %w", jf.Name, out, err)
			}
			fn.Outputs = append(fn.Outputs, td)
		}
		spec.Functions = append(spec.Functions, fn)
	}

	for _, js2 := range js.Structs {
		s := xdr.ScSpecUdtStructV0{Name: js2.Name, Doc: js2.Doc}
		for _, f := range js2.Fields {
			td, err := parseTypeDef(f.Type)
			if err != nil {
				return nil, fmt.Errorf("struct %q field %q: %w", js2.Name, f.Name, err)
			}
			s.Fields = append(s.Fields, xdr.ScSpecUdtStructFieldV0{Name: f.Name, Type: td})
		}
		spec.Structs = append(spec.Structs, s)
	}

	for _, je := range js.Enums {
		e := xdr.ScSpecUdtEnumV0{Name: je.Name, Doc: je.Doc}
		for _, c := range je.Cases {
			e.Cases = append(e.Cases, xdr.ScSpecUdtEnumCaseV0{
				Name:  c.Name,
				Value: xdr.Uint32(c.Value),
			})
		}
		spec.Enums = append(spec.Enums, e)
	}

	for _, ju := range js.Unions {
		u := xdr.ScSpecUdtUnionV0{Name: ju.Name, Doc: ju.Doc}
		for _, c := range ju.Cases {
			if len(c.Types) == 0 {
				u.Cases = append(u.Cases, xdr.ScSpecUdtUnionCaseV0{
					Kind:     xdr.ScSpecUdtUnionCaseV0KindScSpecUdtUnionCaseVoidV0,
					VoidCase: &xdr.ScSpecUdtUnionCaseVoidV0{Name: c.Name},
				})
			} else {
				types := make([]xdr.ScSpecTypeDef, len(c.Types))
				for i, t := range c.Types {
					td, err := parseTypeDef(t)
					if err != nil {
						return nil, fmt.Errorf("union %q case %q type[%d]: %w", ju.Name, c.Name, i, err)
					}
					types[i] = td
				}
				u.Cases = append(u.Cases, xdr.ScSpecUdtUnionCaseV0{
					Kind: xdr.ScSpecUdtUnionCaseV0KindScSpecUdtUnionCaseTupleV0,
					TupleCase: &xdr.ScSpecUdtUnionCaseTupleV0{
						Name: c.Name,
						Type: types,
					},
				})
			}
		}
		spec.Unions = append(spec.Unions, u)
	}

	for _, je := range js.ErrorEnums {
		e := xdr.ScSpecUdtErrorEnumV0{Name: je.Name, Doc: je.Doc}
		for _, c := range je.Cases {
			e.Cases = append(e.Cases, xdr.ScSpecUdtErrorEnumCaseV0{
				Name:  c.Name,
				Value: xdr.Uint32(c.Value),
			})
		}
		spec.ErrorEnums = append(spec.ErrorEnums, e)
	}

	for _, jev := range js.Events {
		ev := xdr.ScSpecEventV0{Name: xdr.ScSymbol(jev.Name), Doc: jev.Doc}
		for _, p := range jev.Params {
			td, err := parseTypeDef(p.Type)
			if err != nil {
				return nil, fmt.Errorf("event %q param %q: %w", jev.Name, p.Name, err)
			}
			loc := xdr.ScSpecEventParamLocationV0ScSpecEventParamLocationData
			if p.Location == "topic" {
				loc = xdr.ScSpecEventParamLocationV0ScSpecEventParamLocationTopicList
			}
			ev.Params = append(ev.Params, xdr.ScSpecEventParamV0{
				Name:     p.Name,
				Type:     td,
				Location: loc,
			})
		}
		spec.Events = append(spec.Events, ev)
	}

	return spec, nil
}

// parseTypeDef converts a human-readable type string (as produced by
// FormatTypeDef) back into an xdr.ScSpecTypeDef.
func parseTypeDef(s string) (xdr.ScSpecTypeDef, error) { //nolint:cyclop
	switch s {
	case "Val":
		return xdr.ScSpecTypeDef{Type: xdr.ScSpecTypeScSpecTypeVal}, nil
	case "Bool":
		return xdr.ScSpecTypeDef{Type: xdr.ScSpecTypeScSpecTypeBool}, nil
	case "Void":
		return xdr.ScSpecTypeDef{Type: xdr.ScSpecTypeScSpecTypeVoid}, nil
	case "Error":
		return xdr.ScSpecTypeDef{Type: xdr.ScSpecTypeScSpecTypeError}, nil
	case "U32":
		return xdr.ScSpecTypeDef{Type: xdr.ScSpecTypeScSpecTypeU32}, nil
	case "I32":
		return xdr.ScSpecTypeDef{Type: xdr.ScSpecTypeScSpecTypeI32}, nil
	case "U64":
		return xdr.ScSpecTypeDef{Type: xdr.ScSpecTypeScSpecTypeU64}, nil
	case "I64":
		return xdr.ScSpecTypeDef{Type: xdr.ScSpecTypeScSpecTypeI64}, nil
	case "Timepoint":
		return xdr.ScSpecTypeDef{Type: xdr.ScSpecTypeScSpecTypeTimepoint}, nil
	case "Duration":
		return xdr.ScSpecTypeDef{Type: xdr.ScSpecTypeScSpecTypeDuration}, nil
	case "U128":
		return xdr.ScSpecTypeDef{Type: xdr.ScSpecTypeScSpecTypeU128}, nil
	case "I128":
		return xdr.ScSpecTypeDef{Type: xdr.ScSpecTypeScSpecTypeI128}, nil
	case "U256":
		return xdr.ScSpecTypeDef{Type: xdr.ScSpecTypeScSpecTypeU256}, nil
	case "I256":
		return xdr.ScSpecTypeDef{Type: xdr.ScSpecTypeScSpecTypeI256}, nil
	case "Bytes":
		return xdr.ScSpecTypeDef{Type: xdr.ScSpecTypeScSpecTypeBytes}, nil
	case "String":
		return xdr.ScSpecTypeDef{Type: xdr.ScSpecTypeScSpecTypeString}, nil
	case "Symbol":
		return xdr.ScSpecTypeDef{Type: xdr.ScSpecTypeScSpecTypeSymbol}, nil
	case "Address":
		return xdr.ScSpecTypeDef{Type: xdr.ScSpecTypeScSpecTypeAddress}, nil
	case "MuxedAddress":
		return xdr.ScSpecTypeDef{Type: xdr.ScSpecTypeScSpecTypeMuxedAddress}, nil
	}

	// Parameterised types: Option<T>, Vec<T>, Map<K,V>, Result<T,E>, BytesN(N), (T1,T2,...), UDT
	if inner, ok := stripWrapper(s, "Option<", ">"); ok {
		td, err := parseTypeDef(inner)
		if err != nil {
			return xdr.ScSpecTypeDef{}, fmt.Errorf("Option inner: %w", err)
		}
		return xdr.ScSpecTypeDef{
			Type:   xdr.ScSpecTypeScSpecTypeOption,
			Option: &xdr.ScSpecTypeOption{ValueType: td},
		}, nil
	}

	if inner, ok := stripWrapper(s, "Vec<", ">"); ok {
		td, err := parseTypeDef(inner)
		if err != nil {
			return xdr.ScSpecTypeDef{}, fmt.Errorf("Vec inner: %w", err)
		}
		return xdr.ScSpecTypeDef{
			Type: xdr.ScSpecTypeScSpecTypeVec,
			Vec:  &xdr.ScSpecTypeVec{ElementType: td},
		}, nil
	}

	if inner, ok := stripWrapper(s, "Map<", ">"); ok {
		k, v, err := splitTwoTypes(inner)
		if err != nil {
			return xdr.ScSpecTypeDef{}, fmt.Errorf("Map types: %w", err)
		}
		ktd, err := parseTypeDef(k)
		if err != nil {
			return xdr.ScSpecTypeDef{}, fmt.Errorf("Map key: %w", err)
		}
		vtd, err := parseTypeDef(v)
		if err != nil {
			return xdr.ScSpecTypeDef{}, fmt.Errorf("Map value: %w", err)
		}
		return xdr.ScSpecTypeDef{
			Type: xdr.ScSpecTypeScSpecTypeMap,
			Map:  &xdr.ScSpecTypeMap{KeyType: ktd, ValueType: vtd},
		}, nil
	}

	if inner, ok := stripWrapper(s, "Result<", ">"); ok {
		ok2, errT, err := splitTwoTypes(inner)
		if err != nil {
			return xdr.ScSpecTypeDef{}, fmt.Errorf("Result types: %w", err)
		}
		oktd, err := parseTypeDef(ok2)
		if err != nil {
			return xdr.ScSpecTypeDef{}, fmt.Errorf("Result ok: %w", err)
		}
		errtd, err := parseTypeDef(errT)
		if err != nil {
			return xdr.ScSpecTypeDef{}, fmt.Errorf("Result error: %w", err)
		}
		return xdr.ScSpecTypeDef{
			Type:   xdr.ScSpecTypeScSpecTypeResult,
			Result: &xdr.ScSpecTypeResult{OkType: oktd, ErrorType: errtd},
		}, nil
	}

	if inner, ok := stripWrapper(s, "BytesN(", ")"); ok {
		var n uint32
		if _, err := fmt.Sscanf(inner, "%d", &n); err != nil {
			return xdr.ScSpecTypeDef{}, fmt.Errorf("BytesN value: %w", err)
		}
		return xdr.ScSpecTypeDef{
			Type:   xdr.ScSpecTypeScSpecTypeBytesN,
			BytesN: &xdr.ScSpecTypeBytesN{N: xdr.Uint32(n)},
		}, nil
	}

	if inner, ok := stripWrapper(s, "(", ")"); ok {
		parts, err := splitTypes(inner)
		if err != nil {
			return xdr.ScSpecTypeDef{}, fmt.Errorf("Tuple types: %w", err)
		}
		types := make([]xdr.ScSpecTypeDef, len(parts))
		for i, p := range parts {
			types[i], err = parseTypeDef(p)
			if err != nil {
				return xdr.ScSpecTypeDef{}, fmt.Errorf("Tuple[%d]: %w", i, err)
			}
		}
		return xdr.ScSpecTypeDef{
			Type:  xdr.ScSpecTypeScSpecTypeTuple,
			Tuple: &xdr.ScSpecTypeTuple{ValueTypes: types},
		}, nil
	}

	// Treat anything else as a UDT name.
	if s != "" {
		return xdr.ScSpecTypeDef{
			Type: xdr.ScSpecTypeScSpecTypeUdt,
			Udt:  &xdr.ScSpecTypeUdt{Name: s},
		}, nil
	}

	return xdr.ScSpecTypeDef{}, fmt.Errorf("unrecognised type %q", s)
}

// stripWrapper returns the inner string if s starts with prefix and ends with
// suffix, otherwise returns ("", false).
func stripWrapper(s, prefix, suffix string) (string, bool) {
	if len(s) >= len(prefix)+len(suffix) &&
		s[:len(prefix)] == prefix &&
		s[len(s)-len(suffix):] == suffix {
		return s[len(prefix) : len(s)-len(suffix)], true
	}
	return "", false
}

// splitTwoTypes splits a comma-separated pair of type strings, respecting
// angle-bracket nesting (e.g. "Map<K, V>, U32" → ["Map<K, V>", "U32"]).
func splitTwoTypes(s string) (string, string, error) {
	parts, err := splitTypes(s)
	if err != nil {
		return "", "", err
	}
	if len(parts) != 2 {
		return "", "", fmt.Errorf("expected 2 types, got %d in %q", len(parts), s)
	}
	return parts[0], parts[1], nil
}

// splitTypes splits a comma-separated list of type strings, respecting
// angle-bracket and parenthesis nesting.
func splitTypes(s string) ([]string, error) {
	var parts []string
	depth := 0
	start := 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '<', '(':
			depth++
		case '>', ')':
			depth--
		case ',':
			if depth == 0 {
				parts = append(parts, trimSpace(s[start:i]))
				start = i + 1
			}
		}
	}
	if depth != 0 {
		return nil, fmt.Errorf("unbalanced brackets in %q", s)
	}
	last := trimSpace(s[start:])
	if last != "" {
		parts = append(parts, last)
	}
	return parts, nil
}

func trimSpace(s string) string {
	start, end := 0, len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t') {
		end--
	}
	return s[start:end]
}
