// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package v1

import (
	"encoding/json"
	"errors"
	"fmt"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

// MarshalReferrer encodes c into a RecordReferrer.
func (c *IdentityClaim) MarshalReferrer() (*corev1.RecordReferrer, error) {
	if c == nil {
		return nil, errors.New("identity claim is nil")
	}

	data, err := marshalReferrerData(c)
	if err != nil {
		return nil, err
	}

	return &corev1.RecordReferrer{Type: corev1.IdentityClaimReferrerType, Data: data}, nil
}

// UnmarshalReferrer loads c from a RecordReferrer.
func (c *IdentityClaim) UnmarshalReferrer(ref *corev1.RecordReferrer) error {
	return unmarshalReferrerData(ref, c)
}

// MarshalReferrer encodes c into a RecordReferrer.
func (c *OwnershipClaim) MarshalReferrer() (*corev1.RecordReferrer, error) {
	if c == nil {
		return nil, errors.New("ownership claim is nil")
	}

	data, err := marshalReferrerData(c)
	if err != nil {
		return nil, err
	}

	return &corev1.RecordReferrer{Type: corev1.OwnershipClaimReferrerType, Data: data}, nil
}

// UnmarshalReferrer loads c from a RecordReferrer.
func (c *OwnershipClaim) UnmarshalReferrer(ref *corev1.RecordReferrer) error {
	return unmarshalReferrerData(ref, c)
}

// marshalReferrerData converts a proto message to a RecordReferrer's Data field.
func marshalReferrerData(m proto.Message) (*structpb.Struct, error) {
	jsonBytes, err := protojson.Marshal(m)
	if err != nil {
		return nil, fmt.Errorf("marshal claim to json: %w", err)
	}

	var raw map[string]any
	if err := json.Unmarshal(jsonBytes, &raw); err != nil {
		return nil, fmt.Errorf("unmarshal json to map: %w", err)
	}

	data, err := structpb.NewStruct(raw)
	if err != nil {
		return nil, fmt.Errorf("convert map to struct: %w", err)
	}

	return data, nil
}

// unmarshalReferrerData loads ref's Data field into a proto message.
func unmarshalReferrerData(ref *corev1.RecordReferrer, m proto.Message) error {
	if ref == nil || ref.GetData() == nil {
		return errors.New("referrer or data is nil")
	}

	jsonBytes, err := json.Marshal(ref.GetData().AsMap())
	if err != nil {
		return fmt.Errorf("marshal struct to json: %w", err)
	}

	if err := protojson.Unmarshal(jsonBytes, m); err != nil {
		return fmt.Errorf("unmarshal json to claim: %w", err)
	}

	return nil
}
