// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package v1

import (
	"errors"
	"fmt"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
)

// ReferrerType returns the referrer type for Verification.
func (v *Verification) ReferrerType() string {
	return string((&Verification{}).ProtoReflect().Descriptor().FullName())
}

// MarshalReferrer exports the Verification into a RecordReferrer.
func (v *Verification) MarshalReferrer() (*corev1.RecordReferrer, error) {
	if v == nil {
		return nil, errors.New("verification is nil")
	}

	// Use protojson to marshal the message to JSON bytes
	jsonBytes, err := protojson.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal verification to JSON: %w", err)
	}

	// Convert JSON to structpb.Struct
	data := &structpb.Struct{}
	if err := protojson.Unmarshal(jsonBytes, data); err != nil {
		return nil, fmt.Errorf("failed to convert JSON to struct: %w", err)
	}

	return &corev1.RecordReferrer{
		Type: v.ReferrerType(),
		Data: data,
	}, nil
}

// UnmarshalReferrer loads the Verification from a RecordReferrer.
func (v *Verification) UnmarshalReferrer(ref *corev1.RecordReferrer) error {
	if ref == nil || ref.GetData() == nil {
		return errors.New("referrer or data is nil")
	}

	// Convert structpb.Struct back to JSON bytes
	jsonBytes, err := protojson.Marshal(ref.GetData())
	if err != nil {
		return fmt.Errorf("failed to marshal struct to JSON: %w", err)
	}

	// Use protojson to unmarshal into the Verification message
	// This properly handles oneof fields
	if err := protojson.Unmarshal(jsonBytes, v); err != nil {
		return fmt.Errorf("failed to unmarshal verification from JSON: %w", err)
	}

	return nil
}

// NewDomainVerification creates a new Verification with DomainVerification info.
func NewDomainVerification(dv *DomainVerification) *Verification {
	return &Verification{
		Info: &Verification_Domain{
			Domain: dv,
		},
	}
}
