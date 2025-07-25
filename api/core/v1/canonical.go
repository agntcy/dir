// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package corev1

import "errors"

// MarshalCanonical marshals the OASF object inside the record using canonical JSON serialization.
// This ensures deterministic, cross-language compatible byte representation.
// The output represents the pure OASF object data and is used for both CID calculation and storage.
func (r *Record) MarshalCanonical() ([]byte, error) {
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
func UnmarshalCanonical(data []byte) (*Record, error) {
	return LoadOASFFromBytes(data)
}
