// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package v1

import (
	"errors"
	"fmt"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/oasf-sdk/pkg/decoder"
)

// ReferrerType returns the referrer type for PublicKey.
func (p *PublicKey) ReferrerType() string {
	return string((&PublicKey{}).ProtoReflect().Descriptor().FullName())
}

// MarshalReferrer exports the PublicKey into a RecordReferrer.
func (p *PublicKey) MarshalReferrer() (*corev1.RecordReferrer, error) {
	if p == nil {
		return nil, errors.New("public key is nil")
	}

	// Use decoder to convert proto message to structpb
	data, err := decoder.StructToProto(p)
	if err != nil {
		return nil, fmt.Errorf("failed to convert public key to struct: %w", err)
	}

	return &corev1.RecordReferrer{
		Type: p.ReferrerType(),
		Data: data,
	}, nil
}

// UnmarshalReferrer loads the PublicKey from a RecordReferrer.
func (p *PublicKey) UnmarshalReferrer(ref *corev1.RecordReferrer) error {
	if ref == nil || ref.GetData() == nil {
		return errors.New("referrer or data is nil")
	}

	// Use decoder to convert structpb to proto message
	decoded, err := decoder.ProtoToStruct[PublicKey](ref.GetData())
	if err != nil {
		return fmt.Errorf("failed to decode public key from referrer: %w", err)
	}

	// Copy fields individually to avoid copying the lock
	p.Key = decoded.GetKey()

	return nil
}
