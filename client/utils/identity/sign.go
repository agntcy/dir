// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package identity signs and verifies identity.v1 Claims on top of the
// payload-agnostic client/utils/jws package.
package identity

import (
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	identityv1 "github.com/agntcy/dir/api/identity/v1"
	"github.com/agntcy/dir/client/utils/jws"
)

// SignOption customizes Sign.
type SignOption func(*signOptions)

type signOptions struct {
	subjectChecker      func(subject string) bool
	certificateProvider func() []byte
}

// WithSubjectChecker makes Sign fail unless check accepts the claim's
// subject, e.g. jws.KeyCertSigner.SubjectMatchesCertificate.
func WithSubjectChecker(check func(subject string) bool) SignOption {
	return func(o *signOptions) { o.subjectChecker = check }
}

// WithCertificateProvider makes Sign embed the DER certificate returned by
// provider in the claim, e.g. jws.KeyCertSigner.CertificateDER.
func WithCertificateProvider(provider func() []byte) SignOption {
	return func(o *signOptions) { o.certificateProvider = provider }
}

// Sign signs claim for recordCID using signer, setting RecordCid, SignedAt
// and Signature, plus Certificate when WithCertificateProvider is given.
func Sign(claim *identityv1.Claim, recordCID string, signer jws.Signer, opts ...SignOption) error {
	if claim == nil {
		return errors.New("claim is nil")
	}

	if claim.GetRole() == identityv1.ClaimRole_CLAIM_ROLE_UNSPECIFIED {
		return errors.New("role is required")
	}

	subject := claim.GetSubject()
	if subject == "" {
		return errors.New("subject is required")
	}

	if signer == nil {
		return errors.New("signer is required")
	}

	var o signOptions
	for _, opt := range opts {
		opt(&o)
	}

	if o.subjectChecker != nil && !o.subjectChecker(subject) {
		return fmt.Errorf("signer does not cover claimed subject %q", subject)
	}

	claim.RecordCid = recordCID
	claim.SignedAt = time.Now().UTC().Format(time.RFC3339)

	payload, err := claim.GetPayload()
	if err != nil {
		return fmt.Errorf("get claim payload: %w", err)
	}

	sig, err := signer.Sign(payload)
	if err != nil {
		return fmt.Errorf("sign claim: %w", err)
	}

	claim.Signature = sig

	if o.certificateProvider != nil {
		cert := base64.StdEncoding.EncodeToString(o.certificateProvider())
		claim.Certificate = &cert
	}

	return nil
}
