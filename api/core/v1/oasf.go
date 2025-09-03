// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package corev1

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"

	objectsv1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/objects/v1"
	objectsv2 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/objects/v2"
	objectsv3 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/objects/v3"
)

// DetectOASFVersion detects the OASF schema version from JSON data.
func detectOASFVersion(data []byte) (string, error) {
	// VersionDetector is used to detect OASF schema version from JSON data.
	type VersionDetector struct {
		SchemaVersion string `json:"schema_version"`
	}

	var detector VersionDetector

	err := json.Unmarshal(data, &detector)
	if err != nil {
		return "", fmt.Errorf("failed to parse JSON for version detection: %w", err)
	}

	if detector.SchemaVersion == "" {
		// If the schema version is not set, we need to throw an error
		return "", errors.New("schema version is not set")
	}

	return detector.SchemaVersion, nil
}

// LoadOASFFromReader loads OASF data from reader and returns a Record with proper version detection.
func LoadOASFFromReader(reader io.Reader) (*Record, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read data: %w", err)
	}

	return loadOASFFromBytes(data)
}

// LoadOASFFromBytes loads OASF data from bytes and returns a Record with proper version detection.
// This is the core function used by the new UnmarshalCanonical implementation.
func loadOASFFromBytes(data []byte) (*Record, error) {
	version, err := detectOASFVersion(data)
	if err != nil {
		return nil, err
	}

	switch version {
	case "v0.3.1":
		agent := &objectsv1.Agent{}

		err := json.Unmarshal(data, agent)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal v0.3.1 Agent: %w", err)
		}

		return &Record{Data: &Record_V1{V1: agent}}, nil

	case "v0.4.0":
		agentRecord := &objectsv2.AgentRecord{}

		err := json.Unmarshal(data, agentRecord)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal v0.4.0 AgentRecord: %w", err)
		}

		return &Record{Data: &Record_V2{V2: agentRecord}}, nil

	case "v0.5.0":
		record := &objectsv3.Record{}

		err := json.Unmarshal(data, record)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal v0.5.0 Record: %w", err)
		}

		return &Record{Data: &Record_V3{V3: record}}, nil

	default:
		return nil, fmt.Errorf("unsupported OASF version: %s (supported: v0.3.1, v0.4.0, v0.5.0)", version)
	}
}

// marshalOASFCanonical marshals an OASF object using canonical JSON serialization.
// This ensures deterministic, cross-language compatible byte representation.
// Uses regular json.Marshal to match the format users work with in the rest of the codebase.
func marshalOASFCanonical(oasfObject interface{}) ([]byte, error) {
	if oasfObject == nil {
		return nil, nil
	}

	// Step 1: Convert to JSON using regular json.Marshal (consistent with cli/cmd/pull and LoadOASFFromBytes)
	jsonBytes, err := json.Marshal(oasfObject)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal OASF object to JSON: %w", err)
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

// MarshalCanonical marshals the OASF object inside the record using canonical JSON serialization.
// This ensures deterministic, cross-language compatible byte representation.
// The output represents the pure OASF object data and is used for both CID calculation and storage.
func (r *Record) MarshalOASF() ([]byte, error) {
	if r == nil {
		return nil, nil
	}

	// Extract the OASF object based on version and marshal it canonically
	// Use regular JSON marshaling to match the format users work with
	switch data := r.GetData().(type) {
	case *Record_V1:
		return marshalOASFCanonical(data.V1)
	case *Record_V2:
		return marshalOASFCanonical(data.V2)
	case *Record_V3:
		return marshalOASFCanonical(data.V3)
	default:
		return nil, errors.New("unsupported record type")
	}
}

// UnmarshalCanonical unmarshals canonical OASF object JSON bytes to a Record.
// This function detects the OASF version from the data and constructs the appropriate Record wrapper.
func UnmarshalOASF(data []byte) (*Record, error) {
	return loadOASFFromBytes(data)
}
