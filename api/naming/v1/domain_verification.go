// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package v1

import (
	"errors"
	"fmt"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/oasf-sdk/pkg/decoder"
)

// ReferrerType returns the referrer type for DomainVerification.
func (d *DomainVerification) ReferrerType() string {
	return string((&DomainVerification{}).ProtoReflect().Descriptor().FullName())
}

// MarshalReferrer exports the DomainVerification into a RecordReferrer.
func (d *DomainVerification) MarshalReferrer() (*corev1.RecordReferrer, error) {
	if d == nil {
		return nil, errors.New("domain verification is nil")
	}

	// Use decoder to convert proto message to structpb
	data, err := decoder.StructToProto(d)
	if err != nil {
		return nil, fmt.Errorf("failed to convert domain verification to struct: %w", err)
	}

	return &corev1.RecordReferrer{
		Type: d.ReferrerType(),
		Data: data,
	}, nil
}

// UnmarshalReferrer loads the DomainVerification from a RecordReferrer.
func (d *DomainVerification) UnmarshalReferrer(ref *corev1.RecordReferrer) error {
	if ref == nil || ref.GetData() == nil {
		return errors.New("referrer or data is nil")
	}

	// Use decoder to convert structpb to proto message
	decoded, err := decoder.ProtoToStruct[DomainVerification](ref.GetData())
	if err != nil {
		return fmt.Errorf("failed to decode domain verification from referrer: %w", err)
	}

	// Copy fields individually to avoid copying the lock
	d.Domain = decoded.GetDomain()
	d.Method = decoded.GetMethod()
	d.MatchedKeyId = decoded.GetMatchedKeyId()
	d.VerifiedAt = decoded.GetVerifiedAt()

	return nil
}
