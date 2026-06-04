// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package v1

import (
	"errors"
	"fmt"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/oasf-sdk/pkg/decoder"
)

// ReferrerType returns the referrer type for OwnershipClaim.
func (c *OwnershipClaim) ReferrerType() string {
	return corev1.OwnershipClaimReferrerType
}

// MarshalReferrer exports the OwnershipClaim into a RecordReferrer.
func (c *OwnershipClaim) MarshalReferrer() (*corev1.RecordReferrer, error) {
	if c == nil {
		return nil, errors.New("ownership claim is nil")
	}

	data, err := decoder.StructToProto(c)
	if err != nil {
		return nil, fmt.Errorf("failed to convert ownership claim to struct: %w", err)
	}

	return &corev1.RecordReferrer{
		Type: c.ReferrerType(),
		Data: data,
	}, nil
}

// UnmarshalReferrer loads the OwnershipClaim from a RecordReferrer.
func (c *OwnershipClaim) UnmarshalReferrer(ref *corev1.RecordReferrer) error {
	if ref == nil || ref.GetData() == nil {
		return errors.New("referrer or data is nil")
	}

	decoded, err := decoder.ProtoToStruct[OwnershipClaim](ref.GetData())
	if err != nil {
		return fmt.Errorf("failed to decode ownership claim from referrer: %w", err)
	}

	c.OwnerId = decoded.GetOwnerId()
	c.ClaimedAt = decoded.GetClaimedAt()

	return nil
}
