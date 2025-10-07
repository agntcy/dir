// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package v1

import (
	"errors"
	"fmt"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/oasf-sdk/pkg/decoder"
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

	// Use decoder to convert proto message to structpb
	data, err := decoder.StructToProto(s)
	if err != nil {
		return nil, fmt.Errorf("failed to convert signature to struct: %w", err)
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

	// Use decoder to convert structpb to proto message
	decoded, err := decoder.ProtoToStruct[Signature](ref.GetData())
	if err != nil {
		return fmt.Errorf("failed to decode signature from referrer: %w", err)
	}

	// Copy fields individually to avoid copying the lock
	s.Annotations = decoded.GetAnnotations()
	s.SignedAt = decoded.GetSignedAt()
	s.Algorithm = decoded.GetAlgorithm()
	s.Signature = decoded.GetSignature()
	s.Certificate = decoded.GetCertificate()
	s.ContentType = decoded.GetContentType()
	s.ContentBundle = decoded.GetContentBundle()

	return nil
}
