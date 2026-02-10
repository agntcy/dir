// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package v1

import (
	"encoding/json"
	"errors"
	"fmt"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
)

// ReferrerType returns the type for Signature.
func (s *Signature) ReferrerType() string {
	return string((&Signature{}).ProtoReflect().Descriptor().FullName())
}

// MarshalReferrer exports the Signature into a RecordReferrer.
func (s *Signature) MarshalReferrer() (*corev1.RecordReferrer, error) {
	if s == nil {
		return nil, errors.New("signature is nil")
	}

	// Marshal signature to JSON using protojson
	jsonBytes, err := protojson.Marshal(s)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal signature to json: %w", err)
	}

	// Unmarshal JSON to a map
	var m map[string]any
	if err := json.Unmarshal(jsonBytes, &m); err != nil {
		return nil, fmt.Errorf("failed to unmarshal json to map: %w", err)
	}

	// Convert map to structpb.Struct
	data, err := structpb.NewStruct(m)
	if err != nil {
		return nil, fmt.Errorf("failed to convert map to struct: %w", err)
	}

	return &corev1.RecordReferrer{
		Type: s.ReferrerType(),
		Data: data,
	}, nil
}

// UnmarshalReferrer loads the Signature from a RecordReferrer.
func (s *Signature) UnmarshalReferrer(ref *corev1.RecordReferrer) error {
	if ref == nil || ref.GetData() == nil {
		return errors.New("referrer or data is nil")
	}

	// Convert structpb.Struct to JSON
	jsonBytes, err := json.Marshal(ref.GetData().AsMap())
	if err != nil {
		return fmt.Errorf("failed to marshal struct to json: %w", err)
	}

	// Unmarshal JSON to Signature using protojson
	if err := protojson.Unmarshal(jsonBytes, s); err != nil {
		return fmt.Errorf("failed to unmarshal json to signature: %w", err)
	}

	return nil
}
