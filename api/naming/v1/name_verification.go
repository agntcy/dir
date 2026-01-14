// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package v1

import (
	"errors"
	"fmt"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/oasf-sdk/pkg/decoder"
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

	// Use decoder to convert proto message to structpb.
	data, err := decoder.StructToProto(v)
	if err != nil {
		return nil, fmt.Errorf("failed to convert verification to struct: %w", err)
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

	// Use decoder to convert structpb to proto message.
	decoded, err := decoder.ProtoToStruct[Verification](ref.GetData())
	if err != nil {
		return fmt.Errorf("failed to decode verification from referrer: %w", err)
	}

	// Copy the oneof field.
	v.Info = decoded.GetInfo()

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
