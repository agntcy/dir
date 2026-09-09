// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package v1

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/agntcy/dir/api/core/adapters"
	"github.com/agntcy/oasf-sdk/pkg/decoder"
	ocidigest "github.com/opencontainers/go-digest"
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

// getAnnotation extracts a value from the record's top-level "annotations" map field.
func (r *Record) getAnnotation(key string) string {
	if r == nil || r.GetData() == nil {
		return ""
	}

	annotations, ok := r.GetData().GetFields()["annotations"]
	if !ok {
		return ""
	}

	if v, ok := annotations.GetStructValue().GetFields()[key]; ok {
		return v.GetStringValue()
	}

	return ""
}

// GetIdentity extracts the record's own claimed identity URI from its annotations.
func (r *Record) GetIdentity() string {
	return r.getAnnotation(AnnotationKeyIdentity)
}

// GetIdentityType returns the record identity's URI scheme, using the explicit
// annotation override when present, otherwise inferring it from the URI itself.
func (r *Record) GetIdentityType() string {
	if t := r.getAnnotation(AnnotationKeyIdentityType); t != "" {
		return t
	}

	if identity := r.GetIdentity(); identity != "" {
		return InferIdentityType(identity)
	}

	return ""
}

// GetOwner extracts the record's claimed owner identity URI from its annotations.
func (r *Record) GetOwner() string {
	return r.getAnnotation(AnnotationKeyOwner)
}

// GetOwnerType returns the owner identity's URI scheme, using the explicit
// annotation override when present, otherwise inferring it from the URI itself.
func (r *Record) GetOwnerType() string {
	if t := r.getAnnotation(AnnotationKeyOwnerType); t != "" {
		return t
	}

	if owner := r.GetOwner(); owner != "" {
		return InferIdentityType(owner)
	}

	return ""
}

// InferIdentityType infers an identity URI's scheme from its prefix.
// Returns "did", "spiffe", "https", or "dns" (the fallback for bare domains
// and explicit "dns:" URIs).
func InferIdentityType(uri string) string {
	switch {
	case strings.HasPrefix(uri, "did:"):
		return "did"
	case strings.HasPrefix(uri, "spiffe://"):
		return "spiffe"
	case strings.HasPrefix(uri, "https://"), strings.HasPrefix(uri, "http://"):
		return "https"
	default:
		return "dns"
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

	digest, err := r.contentDigest()
	if err != nil {
		return ""
	}

	cid, err := ConvertDigestToCID(digest)
	if err != nil {
		return ""
	}

	return cid
}

// GetDigest returns the record's content digest in "algorithm:hex" form
// (e.g. "sha256:9f86d0..."), matching common external conventions such as
// OCI digests and the ai-catalog.io Trust Manifest's subject.digest field.
// Computed from the same canonical bytes as GetCid, just a different
// encoding of the same digest - this is purely an interop accessor and does
// not change how records are addressed/stored internally.
// Returns empty string if calculation fails.
func (r *Record) GetDigest() string {
	if r == nil || r.GetData() == nil {
		return ""
	}

	digest, err := r.contentDigest()
	if err != nil {
		return ""
	}

	return digest.String()
}

// contentDigest returns the OCI-style digest of the record's canonical
// bytes, shared by GetCid and GetDigest.
func (r *Record) contentDigest() (ocidigest.Digest, error) {
	canonicalBytes, err := r.Marshal()
	if err != nil {
		return "", err
	}

	return CalculateDigest(canonicalBytes)
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

	// Get CID for adapter
	cid := r.GetCid()
	if cid == "" {
		return nil, fmt.Errorf("failed to calculate CID for record")
	}

	// Create adapter based on record type
	adapter, err := adapters.GetRecordAdapter(cid, decoded)
	if err != nil {
		return nil, fmt.Errorf("failed to create Record adapter: %w", err)
	}

	// Wrap in our DecodedRecord interface
	return &decodedRecord{
		DecodeRecordResponse: decoded,
		Record:               adapter,
	}, nil
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

// ValidateWith validates the Record's data using the supplied Validator.
//
// Callers construct a concrete validator at the composition root and pass
// it here, avoiding any reliance on package-level globals or initialization
// order.
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

	// If we can decode the record, then it is structurally valid.
	// Loaded record may be syntactically valid but semantically invalid (e.g. missing required fields).
	// We leave full semantic validation to the caller.
	// Decode the record using OASF SDK
	_, err = decoder.DecodeRecord(dataStruct)
	if err != nil {
		return nil, fmt.Errorf("failed to decode Record: %w", err)
	}

	return &Record{
		Data: dataStruct,
	}, nil
}
