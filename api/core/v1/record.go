// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package v1

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/agntcy/oasf-sdk/pkg/decoder"
	"github.com/agntcy/oasf-sdk/pkg/validator"
	"google.golang.org/protobuf/proto"
)

const (
	maxRecordSize = 1024 * 1024 * 4 // 4MB

	// DefaultSchemaURL is the default OASF schema URL for API-based validation.
	DefaultSchemaURL = "https://schema.oasf.outshift.com"

	// DefaultValidationTimeout is the default timeout for API-based validation HTTP calls.
	// This ensures validation doesn't block indefinitely if the OASF server is slow or unreachable.
	DefaultValidationTimeout = 30 * time.Second
)

var (
	defaultValidator     *validator.Validator
	configMu             sync.RWMutex
	schemaURL            = DefaultSchemaURL
	disableAPIValidation = false
	strictValidation     = true
)

func init() {
	var err error

	defaultValidator, err = validator.New()
	if err != nil {
		panic(fmt.Sprintf("failed to initialize OASF-SDK validator: %v", err))
	}
}

// SetSchemaURL configures the schema URL to use for API-based validation.
// This function is thread-safe and can be called concurrently with validation operations.
func SetSchemaURL(url string) {
	configMu.Lock()
	defer configMu.Unlock()

	schemaURL = url
}

// SetDisableAPIValidation configures whether to disable API-based validation.
// When true, embedded schemas will be used instead of the API validator.
// This function is thread-safe and can be called concurrently with validation operations.
func SetDisableAPIValidation(disable bool) {
	configMu.Lock()
	defer configMu.Unlock()

	disableAPIValidation = disable
}

// SetStrictValidation configures whether to use strict validation mode.
// When true, uses strict validation (fails on unknown attributes, deprecated fields, etc.).
// When false, uses lax validation (more permissive, only fails on critical errors).
// This function is thread-safe and can be called concurrently with validation operations.
func SetStrictValidation(strict bool) {
	configMu.Lock()
	defer configMu.Unlock()

	strictValidation = strict
}

// CalculateCid calculates and returns the CID for this record.
// The CID is calculated from the record's content using CIDv1, codec 1, SHA2-256.
// Uses canonical JSON marshaling to ensure consistent, cross-language compatible results.
// Returns empty string if calculation fails.
func (r *Record) CalculateCid() string {
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
	schemaVersion, _ := decoder.GetRecordSchemaVersion(r.GetData())

	return schemaVersion
}

// Decode decodes the Record's data into a concrete type using the OASF SDK.
func (r *Record) Decode() (DecodedRecord, error) {
	if r == nil || r.GetData() == nil {
		return nil, errors.New("record is nil")
	}

	// Decode the record using OASF SDK
	decoded, err := decoder.DecodeRecord(r.GetData())
	if err != nil {
		return nil, fmt.Errorf("failed to decode Record: %w", err)
	}

	// Wrap in our DecodedRecord interface
	return &decodedRecord{
		DecodeRecordResponse: decoded,
	}, nil
}

// Validate validates the Record's data against its embedded schema using the OASF SDK.
func (r *Record) Validate(ctx context.Context) (bool, []string, error) {
	if r == nil || r.GetData() == nil {
		return false, []string{"record is nil"}, nil
	}

	recordSize := proto.Size(r)
	if recordSize > maxRecordSize {
		return false, []string{fmt.Sprintf("record size %d bytes exceeds maximum allowed size of %d bytes (4MB)", recordSize, maxRecordSize)}, nil
	}

	// Validate the record using OASF SDK
	// Read configuration atomically to avoid race conditions
	configMu.RLock()

	currentSchemaURL := schemaURL
	currentDisableAPIValidation := disableAPIValidation
	currentStrictValidation := strictValidation

	configMu.RUnlock()

	// If API validation is not disabled, use API-based validation with configured schema URL
	if !currentDisableAPIValidation {
		// Create a context with timeout for API validation HTTP calls.
		// We use the caller's context as parent so validation respects cancellation,
		// but add our own timeout to prevent hanging if the OASF server is slow/unreachable.
		validationCtx, cancel := context.WithTimeout(ctx, DefaultValidationTimeout)
		defer cancel()

		//nolint:wrapcheck
		return defaultValidator.ValidateRecord(
			validationCtx,
			r.GetData(),
			validator.WithSchemaURL(currentSchemaURL),
			validator.WithStrict(currentStrictValidation),
		)
	}

	// Use embedded schemas (no HTTP calls, so we can use the original context)
	//nolint:wrapcheck
	return defaultValidator.ValidateRecord(ctx, r.GetData())
}

// UnmarshalRecord unmarshals canonical Record JSON bytes to a Record.
func UnmarshalRecord(data []byte) (*Record, error) {
	// Load data from JSON bytes
	dataStruct, err := decoder.JsonToProto(data)
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
