// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package v1

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"crypto/x509"
	corev1 "github.com/agntcy/dir/api/core/v1"
)

// KeyResolver verifies a claim's signature for a non-SPIFFE subject by
// resolving the subject's public key from an external source (a DID
// document, a domain's JWKS well-known file, or a DNS TXT record).
// Implemented by server/identity.Registry.
type KeyResolver interface {
	Verify(ctx context.Context, subject string, jwsCompact, payload []byte) (bool, error)
}

// Result is the outcome of verifying an identity or ownership claim.
type Result struct {
	// Verified is true only if the signature validated successfully.
	Verified bool

	// Subject is the identity/owner URI the claim asserts.
	Subject string

	// SubjectType is the inferred URI scheme: "dns", "https", "did", or "spiffe".
	SubjectType string

	// Error explains why verification failed. Empty when Verified is true.
	Error string
}

// VerifyIdentityClaim verifies that c was validly signed for recordCID and
// that its subject matches expectedSubject (the record's own
// "agntcy.dir/identity" annotation).
func VerifyIdentityClaim(ctx context.Context, c *IdentityClaim, recordCID, expectedSubject string, resolver KeyResolver, trustedCerts []*x509.Certificate) *Result {
	if c == nil {
		return &Result{Error: "claim is nil"}
	}

	return verifyClaim(ctx, c.GetSubject(), c.GetSignedAt(), c.GetExpiresAt(), c.GetSignature(), c.GetCertificate(), recordCID, expectedSubject, resolver, trustedCerts)
}

// VerifyOwnershipClaim verifies that c was validly signed for recordCID and
// that its subject matches expectedSubject (the record's own
// "agntcy.dir/owner" annotation).
func VerifyOwnershipClaim(ctx context.Context, c *OwnershipClaim, recordCID, expectedSubject string, resolver KeyResolver, trustedCerts []*x509.Certificate) *Result {
	if c == nil {
		return &Result{Error: "claim is nil"}
	}

	return verifyClaim(ctx, c.GetSubject(), c.GetSignedAt(), c.GetExpiresAt(), c.GetSignature(), c.GetCertificate(), recordCID, expectedSubject, resolver, trustedCerts)
}

func verifyClaim(
	ctx context.Context,
	subject, signedAt, expiresAt, jwsCompact, certificateB64, recordCID, expectedSubject string,
	resolver KeyResolver,
	trustedCerts []*x509.Certificate,
) *Result {
	result := &Result{Subject: subject, SubjectType: corev1.InferIdentityType(subject)}

	if subject == "" {
		result.Error = "subject is required"

		return result
	}

	if expectedSubject == "" {
		result.Error = "record does not declare an identity/owner annotation matching this claim"

		return result
	}

	if subject != expectedSubject {
		result.Error = fmt.Sprintf("claim subject %q does not match record's declared annotation %q", subject, expectedSubject)

		return result
	}

	if jwsCompact == "" {
		result.Error = "signature is empty"

		return result
	}

	if expiresAt != "" {
		expiry, err := time.Parse(time.RFC3339, expiresAt)
		if err == nil && time.Now().After(expiry) {
			result.Error = "claim has expired"

			return result
		}
	}

	payload := CanonicalBytes(recordCID, subject, signedAt)

	if result.SubjectType == "spiffe" {
		if err := verifySpiffe(certificateB64, subject, payload, jwsCompact, trustedCerts); err != nil {
			result.Error = err.Error()

			return result
		}
	} else {
		if resolver == nil {
			result.Error = "no key resolver configured for subject type " + result.SubjectType

			return result
		}

		ok, err := resolver.Verify(ctx, subject, []byte(jwsCompact), payload)
		if err != nil {
			result.Error = err.Error()

			return result
		}

		if !ok {
			result.Error = "signature does not verify against any resolved key"

			return result
		}
	}

	result.Verified = true

	return result
}

// verifySpiffe verifies a SPIFFE claim's embedded certificate and, using its
// public key, the JWS itself. A trust bundle is required: without one there
// is no anchor beyond the claim's own self-signed (or otherwise
// attacker-suppliable) certificate, which proves nothing about the subject's
// legitimacy.
func verifySpiffe(certificateB64, subject string, payload []byte, jwsCompact string, trustedCerts []*x509.Certificate) error {
	if certificateB64 == "" {
		return errors.New("spiffe subject requires an embedded certificate")
	}

	if len(trustedCerts) == 0 {
		return errors.New("no SPIFFE trust bundle configured for this subject's trust domain")
	}

	certDER, err := base64.StdEncoding.DecodeString(certificateB64)
	if err != nil {
		return fmt.Errorf("decode certificate: %w", err)
	}

	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		return fmt.Errorf("parse certificate: %w", err)
	}

	if err := validateCertSubject(cert, subject); err != nil {
		return err
	}

	pool := x509.NewCertPool()
	for _, ca := range trustedCerts {
		pool.AddCert(ca)
	}

	if _, err := cert.Verify(x509.VerifyOptions{Roots: pool}); err != nil {
		return fmt.Errorf("certificate not trusted: %w", err)
	}

	return VerifyJWS(jwsCompact, cert.PublicKey, payload)
}

// validateCertSubject checks that the certificate has a URI SAN equal to subject.
func validateCertSubject(cert *x509.Certificate, subject string) error {
	for _, u := range cert.URIs {
		if u.String() == subject {
			return nil
		}
	}

	return fmt.Errorf("certificate does not contain a URI SAN matching subject %q", subject)
}
