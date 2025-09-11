// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package v1

import (
	"encoding/json"
	"errors"
	"fmt"

	validationv1 "buf.build/gen/go/agntcy/oasf-sdk/protocolbuffers/go/validation/v1"
	oasfcorev1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/core/v1"
	"github.com/agntcy/oasf-sdk/core/converter"
	"github.com/agntcy/oasf-sdk/validation/validator"
)

var defaultValidator *validator.Validator

func init() {
	var err error

	defaultValidator, err = validator.New()
	if err != nil {
		panic(fmt.Sprintf("failed to initialize OASF-SDK validator: %v", err))
	}
}

// GetCid calculates and returns the CID for this record.
// The CID is calculated from the record's content using CIDv1, codec 1, SHA2-256.
// Uses canonical JSON marshaling to ensure consistent, cross-language compatible results.
// Returns empty string if calculation fails.
func (r *Record) GetCid() string {
	if r == nil || r.GetData() == nil {
		return ""
	}

	// Use canonical marshaling for CID calculation
	canonicalBytes, err := r.Marshal()
	if err != nil {
		return ""
	}

	// Calculate digest using local utilities
	digest, err := CalculateDigest(canonicalBytes)
	if err != nil {
		return ""
	}

	// Convert digest to CID using local utilities
	cid, err := ConvertDigestToCID(digest)
	if err != nil {
		return ""
	}

	return cid
}

// Marshal marshals the Record using canonical JSON serialization.
// This ensures deterministic, cross-language compatible byte representation.
// The output represents the pure Record data and is used for both CID calculation and storage.
func (r *Record) Marshal() ([]byte, error) {
	if r == nil || r.GetData() == nil {
		return nil, nil
	}

	// Extract the data marshal it canonically
	// Use regular JSON marshaling to match the format users work with
	// Step 1: Convert to JSON using regular json.Marshal (consistent with cli/cmd/pull)
	jsonBytes, err := json.Marshal(r.GetData())
	if err != nil {
		return nil, fmt.Errorf("failed to marshal Record: %w", err)
	}

	// Step 2: Parse and re-marshal to ensure deterministic map key ordering.
	// This is critical - maps must have consistent key order for deterministic results.
	var normalized interface{}
	if err := json.Unmarshal(jsonBytes, &normalized); err != nil {
		return nil, fmt.Errorf("failed to normalize JSON for canonical ordering: %w", err)
	}

	// Step 3: Marshal with sorted keys for deterministic output.
	// encoding/json.Marshal sorts map keys alphabetically.
	canonicalBytes, err := json.Marshal(normalized)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal normalized JSON with sorted keys: %w", err)
	}

	return canonicalBytes, nil
}

func (r *Record) GetSchemaVersion() string {
	if r == nil || r.GetData() == nil {
		return ""
	}

	// Get schema version from raw using OASF SDK
	schemaVersion, _ := converter.GetRecordSchemaVersion(&oasfcorev1.Object{
		Data: r.GetData(),
	})

	return schemaVersion
}

// Decode decodes the Record's data into a concrete type using the OASF SDK.
func (r *Record) Decode() (DecodedRecord, error) {
	if r == nil || r.GetData() == nil {
		return nil, errors.New("record is nil")
	}

	// Decode the record using OASF SDK
	decoded, err := converter.DecodeRecord(&oasfcorev1.Object{
		Data: r.GetData(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to decode Record: %w", err)
	}

	// Wrap in our DecodedRecord interface
	return &decodedRecord{
		DecodeRecordResponse: decoded,
	}, nil
}

// Validate validates the Record's data against its embedded schema using the OASF SDK.
func (r *Record) Validate() (bool, []string, error) {
	if r == nil || r.GetData() == nil {
		return false, nil, errors.New("record is nil")
	}

	// Validate the record using OASF SDK
	//nolint:wrapcheck
	return defaultValidator.ValidateRecord(&validationv1.ValidateRecordRequest{
		Record: &oasfcorev1.Object{
			Data: r.GetData(),
		},
	})
}

// UnmarshalRecord unmarshals canonical Record JSON bytes to a Record.
func UnmarshalRecord(data []byte) (*Record, error) {
	// Load data from JSON bytes
	dataStruct, err := converter.JsonToProto(data)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal Record: %w", err)
	}

	// Construct a record
	record := &Record{
		Data: dataStruct,
	}

	// If we can decode the record, then it is structurally valid.
	// Loaded record may be syntactically valid but semantically invalid (e.g. missing required fields).
	// We leave full semantic validation to the caller.
	_, err = record.Decode()
	if err != nil {
		return nil, fmt.Errorf("failed to decode Record: %w", err)
	}

	return record, nil
}
