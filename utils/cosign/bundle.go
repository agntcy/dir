// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package cosign

import (
	"crypto/x509"
	"encoding/asn1"
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/sigstore/sigstore-go/pkg/bundle"
	"github.com/sigstore/sigstore/pkg/cryptoutils"
)

// OIDCInfo contains OIDC identity information extracted from a Fulcio certificate.
type OIDCInfo struct {
	// Issuer is the OIDC issuer URL (e.g., "https://github.com/login/oauth")
	Issuer string

	// Identity is the OIDC subject/identity (e.g., "user@example.com")
	Identity string
}

// ParsedBundle represents a parsed Sigstore bundle with extracted components.
type ParsedBundle struct {
	// Bundle is the parsed sigstore-go bundle
	Bundle *bundle.Bundle

	// Certificate is the signing certificate (if present)
	Certificate *x509.Certificate

	// CertificateChain is the full certificate chain
	CertificateChain []*x509.Certificate

	// OIDCInfo contains extracted OIDC identity information
	OIDCInfo *OIDCInfo

	// Signature is the raw signature bytes
	Signature []byte
}

// ParseBundle parses a Sigstore bundle from JSON.
func ParseBundle(bundleJSON string) (*ParsedBundle, error) {
	if bundleJSON == "" {
		return nil, errors.New("bundle JSON is empty")
	}

	// Parse using sigstore-go
	b := &bundle.Bundle{}
	if err := b.UnmarshalJSON([]byte(bundleJSON)); err != nil {
		return nil, fmt.Errorf("failed to parse bundle: %w", err)
	}

	parsed := &ParsedBundle{
		Bundle: b,
	}

	// Extract signature from message signature
	sigContent, err := b.SignatureContent()
	if err == nil {
		if msgSig, ok := sigContent.(*bundle.MessageSignature); ok {
			parsed.Signature = msgSig.Signature()
		}
	}

	// Extract certificate chain from verification content
	verificationContent, err := b.VerificationContent()
	if err == nil && verificationContent != nil {
		if cert := verificationContent.Certificate(); cert != nil {
			parsed.Certificate = cert
			parsed.CertificateChain = []*x509.Certificate{cert}
		}
	}

	// Extract OIDC info from certificate
	if parsed.Certificate != nil {
		oidcInfo, err := ExtractOIDCInfoFromCert(parsed.Certificate)
		if err != nil {
			// Log but don't fail - certificate might not have OIDC extensions
			parsed.OIDCInfo = nil
		} else {
			parsed.OIDCInfo = oidcInfo
		}
	}

	return parsed, nil
}

// Fulcio certificate extension OIDs for OIDC claims
// See: https://github.com/sigstore/fulcio/blob/main/docs/oid-info.md
var (
	// OID 1.3.6.1.4.1.57264.1.1 - OIDC Issuer
	oidIssuer = asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 57264, 1, 1}

	// OID 1.3.6.1.4.1.57264.1.8 - OIDC Issuer (v2)
	oidIssuerV2 = asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 57264, 1, 8}

	// Subject Alternative Name contains the identity
	// For GitHub Actions: OID 1.3.6.1.4.1.57264.1.9 contains the build signer URI
	oidBuildSignerURI = asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 57264, 1, 9}
)

// ExtractOIDCInfoFromCert extracts OIDC issuer and identity from a Fulcio certificate.
func ExtractOIDCInfoFromCert(cert *x509.Certificate) (*OIDCInfo, error) {
	if cert == nil {
		return nil, errors.New("certificate is nil")
	}

	info := &OIDCInfo{}

	// Extract issuer from extensions
	for _, ext := range cert.Extensions {
		// Check for issuer OID (v1 or v2)
		if ext.Id.Equal(oidIssuer) || ext.Id.Equal(oidIssuerV2) {
			// The value is typically a UTF8String or IA5String
			info.Issuer = string(ext.Value)

			// Try to decode as ASN.1 string if raw bytes don't look like a URL
			if len(info.Issuer) > 0 && info.Issuer[0] < 32 {
				// Likely ASN.1 encoded, try to extract the string
				// Skip the ASN.1 tag and length bytes
				if len(ext.Value) > 2 {
					info.Issuer = string(ext.Value[2:])
				}
			}
		}
	}

	// Extract identity from Subject Alternative Names (SAN)
	// Fulcio puts the OIDC subject in the email SAN or URI SAN
	if len(cert.EmailAddresses) > 0 {
		info.Identity = cert.EmailAddresses[0]
	} else if len(cert.URIs) > 0 {
		info.Identity = cert.URIs[0].String()
	}

	// For GitHub Actions, also check the build signer URI extension
	if info.Identity == "" {
		for _, ext := range cert.Extensions {
			if ext.Id.Equal(oidBuildSignerURI) {
				info.Identity = string(ext.Value)
				if len(info.Identity) > 0 && info.Identity[0] < 32 {
					if len(ext.Value) > 2 {
						info.Identity = string(ext.Value[2:])
					}
				}

				break
			}
		}
	}

	if info.Issuer == "" && info.Identity == "" {
		return nil, errors.New("no OIDC information found in certificate")
	}

	return info, nil
}

// ExtractOIDCInfoFromBundle extracts OIDC info from a Sigstore bundle JSON.
func ExtractOIDCInfoFromBundle(bundleJSON string) (*OIDCInfo, error) {
	parsed, err := ParseBundle(bundleJSON)
	if err != nil {
		return nil, err
	}

	if parsed.OIDCInfo == nil {
		return nil, errors.New("no OIDC information found in bundle")
	}

	return parsed.OIDCInfo, nil
}

// GetCertificatePEM returns the signing certificate as PEM-encoded string.
func (p *ParsedBundle) GetCertificatePEM() (string, error) {
	if p.Certificate == nil {
		return "", errors.New("no certificate in bundle")
	}

	pemBytes, err := cryptoutils.MarshalCertificateToPEM(p.Certificate)
	if err != nil {
		return "", fmt.Errorf("failed to marshal certificate to PEM: %w", err)
	}

	return string(pemBytes), nil
}

// GetCertificateChainPEM returns the full certificate chain as PEM-encoded string.
func (p *ParsedBundle) GetCertificateChainPEM() (string, error) {
	if len(p.CertificateChain) == 0 {
		return "", errors.New("no certificate chain in bundle")
	}

	pemBytes, err := cryptoutils.MarshalCertificatesToPEM(p.CertificateChain)
	if err != nil {
		return "", fmt.Errorf("failed to marshal certificate chain to PEM: %w", err)
	}

	return string(pemBytes), nil
}

// EncodeBundleToBase64 encodes a bundle JSON to base64 for storage.
func EncodeBundleToBase64(bundleJSON string) string {
	return base64.StdEncoding.EncodeToString([]byte(bundleJSON))
}

// DecodeBundleFromBase64 decodes a base64-encoded bundle JSON.
func DecodeBundleFromBase64(encoded string) (string, error) {
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64 bundle: %w", err)
	}

	return string(decoded), nil
}
