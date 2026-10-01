// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package identity signs and verifies identity.v1 Claims on top of the
// payload-agnostic client/utils/jws package.
package identity

import (
	"crypto"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"net/url"
	"slices"
	"time"

	identityv1 "github.com/agntcy/dir/api/identity/v1"
	"github.com/agntcy/dir/client/utils/jws"
)

// SignOption customizes Sign.
type SignOption func(*signOptions)

type signOptions struct {
	certificate    []byte
	hasCertificate bool
}

// WithCertificate embeds the signer's X.509 certificate (PEM or DER) in the
// claim. Sign fails unless the certificate belongs to the signer's key and
// carries the claim's subject as a URI SAN.
func WithCertificate(cert []byte) SignOption {
	return func(o *signOptions) {
		o.certificate = cert
		o.hasCertificate = true
	}
}

// Sign signs claim for recordCID using signer, setting RecordCid, SignedAt
// and Signature, plus Certificate when WithCertificate is given.
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

	var certDER []byte

	if o.hasCertificate {
		der, err := certificateFor(o.certificate, signer.Public(), subject)
		if err != nil {
			return err
		}

		certDER = der
	}

	// Sign a copy so an error leaves the caller's claim untouched.
	signed := &identityv1.Claim{
		Role:      claim.GetRole(),
		RecordCid: recordCID,
		Subject:   subject,
		SignedAt:  time.Now().UTC().Format(time.RFC3339),
		ExpiresAt: claim.ExpiresAt,
	}

	payload, err := signed.GetPayload()
	if err != nil {
		return fmt.Errorf("get claim payload: %w", err)
	}

	sig, err := signer.Sign(payload)
	if err != nil {
		return fmt.Errorf("sign claim: %w", err)
	}

	signed.Signature = sig

	if certDER != nil {
		cert := base64.StdEncoding.EncodeToString(certDER)
		signed.Certificate = &cert
	}

	claim.RecordCid = signed.GetRecordCid()
	claim.SignedAt = signed.GetSignedAt()
	claim.Signature = signed.GetSignature()
	claim.Certificate = signed.Certificate

	return nil
}

// certificateFor returns the DER form of data after checking that the
// certificate belongs to pub and names subject as a URI SAN.
func certificateFor(data []byte, pub crypto.PublicKey, subject string) ([]byte, error) {
	der := data
	if block, _ := pem.Decode(data); block != nil {
		der = block.Bytes
	}

	cert, err := x509.ParseCertificate(der)
	if err != nil {
		return nil, fmt.Errorf("parse certificate: %w", err)
	}

	if eq, ok := pub.(interface{ Equal(crypto.PublicKey) bool }); !ok || !eq.Equal(cert.PublicKey) {
		return nil, errors.New("signer key does not match the certificate's public key")
	}

	if !slices.ContainsFunc(cert.URIs, func(u *url.URL) bool { return u.String() == subject }) {
		return nil, fmt.Errorf("certificate does not cover claimed subject %q", subject)
	}

	return der, nil
}
