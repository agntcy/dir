// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package corev1

import (
	"encoding/json"

	"google.golang.org/protobuf/encoding/protojson"
)

// CanonicalMarshal marshals the record using canonical JSON serialization.
// This ensures deterministic, cross-language compatible byte representation.
// The output is used for both CID calculation and storage to maintain consistency.
func (r *Record) CanonicalMarshal() ([]byte, error) {
	if r == nil {
		return nil, nil
	}

	// Step 1: Convert protobuf to JSON with proper protobuf semantics
	jsonBytes, err := protojson.MarshalOptions{
		Multiline:       false, // Single line
		Indent:          "",    // No indentation
		AllowPartial:    false, // Require all required fields
		UseProtoNames:   true,  // Use proto field names (snake_case) for consistency
		UseEnumNumbers:  true,  // Use enum numbers instead of names for stability
		EmitUnpopulated: false, // Don't emit zero/empty values
	}.Marshal(r)
	if err != nil {
		return nil, err
	}

	// Step 2: Parse and re-marshal to ensure deterministic map key ordering
	// This is critical - maps must have consistent key order for deterministic results
	var normalized interface{}
	if err := json.Unmarshal(jsonBytes, &normalized); err != nil {
		return nil, err
	}

	// Step 3: Marshal with sorted keys for deterministic output
	// encoding/json.Marshal sorts map keys alphabetically
	return json.Marshal(normalized)
}

// CanonicalUnmarshal unmarshals canonical JSON bytes back to a Record.
func CanonicalUnmarshal(data []byte) (*Record, error) {
	var record Record
	err := protojson.UnmarshalOptions{
		AllowPartial:   false,
		DiscardUnknown: false,
	}.Unmarshal(data, &record)
	if err != nil {
		return nil, err
	}
	return &record, nil
}
