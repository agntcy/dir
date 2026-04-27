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
	"google.golang.org/protobuf/types/known/structpb"
)

const (
	maxRecordSize = 1024 * 1024 * 4 // 4MB

	// DefaultValidationTimeout is the default timeout for API-based validation HTTP calls.
	// This ensures validation doesn't block indefinitely if the OASF server is slow or unreachable.
	DefaultValidationTimeout = 30 * time.Second
)

// Validator is the minimal contract Record.ValidateWith depends on.
//
// Its shape mirrors github.com/agntcy/oasf-sdk/pkg/validator.Validator so the
// concrete OASF-SDK validator satisfies this interface without an adapter,
// while keeping api/core/v1 free of any process-wide singletons or runtime
// initialization order.
//
// Callers (server, reconciler, CLI, MCP) construct a concrete validator at
// process startup and pass it explicitly to (*Record).ValidateWith.
type Validator interface {
	// ValidateRecord validates the given record data against the configured schema.
	// It returns whether the record is valid, a slice of error messages, a slice of
	// warning messages, and any transport/server error encountered while validating.
	ValidateRecord(ctx context.Context, data *structpb.Struct) (valid bool, errors []string, warnings []string, err error)
}

// defaultValidator is the package-level fallback used by the deprecated
// (*Record).Validate(ctx) method. It is set by InitializeValidator and is
// intentionally retained only to preserve backward compatibility with
// already-published consumers (notably github.com/agntcy/dir-mcp v1.0.0).
//
// Deprecated: prefer constructing a Validator at the composition root and
// calling (*Record).ValidateWith. defaultValidator and the related globals
// will be removed in a future major version of this module (see issue
// https://github.com/agntcy/dir/issues/856).
var (
	defaultValidator Validator
	defaultValidMu   sync.RWMutex
)

// InitializeValidator configures the package-level default OASF validator
// used by the deprecated (*Record).Validate(ctx) method.
//
// Deprecated: this function exists solely for backward compatibility with
// already-published consumers (notably github.com/agntcy/dir-mcp v1.0.0).
// New code MUST NOT depend on it. Construct a *validator.Validator (or any
// type that implements Validator) at process startup and pass it to
// (*Record).ValidateWith. See https://github.com/agntcy/dir/issues/856.
func InitializeValidator(schemaURL string) error {
	if schemaURL == "" {
		return errors.New("schemaURL is required for OASF validation")
	}

	v, err := validator.New(schemaURL)
	if err != nil {
		return fmt.Errorf("failed to initialize OASF validator: %w", err)
	}

	defaultValidMu.Lock()
	defaultValidator = v
	defaultValidMu.Unlock()

	return nil
}

// getDefaultValidator returns the validator configured via InitializeValidator,
// or nil if it has not been configured.
func getDefaultValidator() Validator {
	defaultValidMu.RLock()
	defer defaultValidMu.RUnlock()

	return defaultValidator
}

// GetName extracts the top-level "name" field from the record's data.
func (r *Record) GetName() string {
	if r == nil || r.GetData() == nil {
		return ""
	}

	if v, ok := r.GetData().GetFields()["name"]; ok {
		return v.GetStringValue()
	}

	return ""
}

// GetVersion extracts the top-level "version" field from the record's data.
func (r *Record) GetVersion() string {
	if r == nil || r.GetData() == nil {
		return ""
	}

	if v, ok := r.GetData().GetFields()["version"]; ok {
		return v.GetStringValue()
	}

	return ""
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
	var normalized any
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

// Validate validates the Record using the package-level default validator
// configured via InitializeValidator.
//
// Deprecated: this method depends on global state and exists solely to
// preserve backward compatibility with already-published consumers
// (notably github.com/agntcy/dir-mcp v1.0.0). New code MUST use
// (*Record).ValidateWith and pass an explicit Validator constructed at the
// composition root. This method will be removed in a future major version
// of this module. See https://github.com/agntcy/dir/issues/856.
func (r *Record) Validate(ctx context.Context) (bool, []string, error) {
	v := getDefaultValidator()
	if v == nil {
		return false, []string{"OASF validator is not initialized; call corev1.InitializeValidator or use Record.ValidateWith"}, nil
	}

	return r.ValidateWith(ctx, v)
}

// ValidateWith validates the Record's data using the supplied Validator.
//
// This is the preferred entry point for record validation: callers
// construct a concrete validator at the composition root and inject it
// here, avoiding any reliance on package-level globals or initialization
// order. See https://github.com/agntcy/dir/issues/856.
func (r *Record) ValidateWith(ctx context.Context, v Validator) (bool, []string, error) {
	if r == nil || r.GetData() == nil {
		return false, []string{"record is nil"}, nil
	}

	if v == nil {
		return false, []string{"validator is nil"}, nil
	}

	recordSize := proto.Size(r)
	if recordSize > maxRecordSize {
		return false, []string{fmt.Sprintf("record size %d bytes exceeds maximum allowed size of %d bytes (4MB)", recordSize, maxRecordSize)}, nil
	}

	// Create a context with timeout for API validation HTTP calls.
	// We use the caller's context as parent so validation respects cancellation,
	// but add our own timeout to prevent hanging if the OASF server is slow/unreachable.
	validationCtx, cancel := context.WithTimeout(ctx, DefaultValidationTimeout)
	defer cancel()

	valid, errs, warnings, err := v.ValidateRecord(validationCtx, r.GetData())
	if err != nil {
		return false, nil, fmt.Errorf("failed to validate record: %w", err)
	}

	// Prefix errors and warnings before combining them
	prefixedErrors := make([]string, len(errs))
	for i, e := range errs {
		prefixedErrors[i] = "ERROR: " + e
	}

	prefixedWarnings := make([]string, len(warnings))
	for i, w := range warnings {
		prefixedWarnings[i] = "WARNING: " + w
	}

	allMessages := make([]string, 0, len(prefixedErrors)+len(prefixedWarnings))
	allMessages = append(allMessages, prefixedErrors...)
	allMessages = append(allMessages, prefixedWarnings...)

	return valid, allMessages, nil
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
