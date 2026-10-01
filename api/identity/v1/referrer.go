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

// MarshalReferrer encodes c into a RecordReferrer.
func (c *Claim) MarshalReferrer() (*corev1.RecordReferrer, error) {
	if c == nil {
		return nil, errors.New("claim is nil")
	}

	jsonBytes, err := protojson.Marshal(c)
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

	referrerType := corev1.OwnershipClaimReferrerType
	if c.GetRole() == ClaimRole_CLAIM_ROLE_IDENTITY {
		referrerType = corev1.IdentityClaimReferrerType
	}

	return &corev1.RecordReferrer{Type: referrerType, Data: data}, nil
}

// UnmarshalReferrer loads c from a RecordReferrer.
func (c *Claim) UnmarshalReferrer(ref *corev1.RecordReferrer) error {
	if ref == nil || ref.GetData() == nil {
		return errors.New("referrer or data is nil")
	}

	jsonBytes, err := json.Marshal(ref.GetData().AsMap())
	if err != nil {
		return fmt.Errorf("marshal struct to json: %w", err)
	}

	if err := protojson.Unmarshal(jsonBytes, c); err != nil {
		return fmt.Errorf("unmarshal json to claim: %w", err)
	}

	return nil
}
