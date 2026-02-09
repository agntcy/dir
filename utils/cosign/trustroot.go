// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package cosign

import (
	"crypto/x509"
	"errors"
	"fmt"
	"time"

	"github.com/sigstore/sigstore-go/pkg/root"
	"github.com/sigstore/sigstore-go/pkg/tuf"
	"github.com/sigstore/sigstore/pkg/cryptoutils"
)

const (
	// SigstorePublicGoodBaseURL is the TUF repository URL for Sigstore public good instance.
	SigstorePublicGoodBaseURL = "https://tuf-repo-cdn.sigstore.dev"

	// SigstoreStagingBaseURL is the TUF repository URL for Sigstore staging instance.
	SigstoreStagingBaseURL = "https://tuf-repo-cdn.sigstage.dev"

	// DefaultTrustRootCacheDuration is the default duration to cache trust root data.
	DefaultTrustRootCacheDuration = 24 * time.Hour
)

// TrustRootConfig contains configuration for creating a custom trust root.
type TrustRootConfig struct {
	// FulcioRootPEM is the Fulcio CA root certificate (PEM-encoded).
	// If empty, uses Sigstore public good instance.
	FulcioRootPEM string

	// RekorPublicKeyPEM is the Rekor public key (PEM-encoded).
	// If empty, uses Sigstore public good instance.
	RekorPublicKeyPEM string

	// TimestampAuthorityRootsPEM are the TSA root certificates (PEM-encoded).
	// If empty, uses Sigstore public good instance.
	TimestampAuthorityRootsPEM []string

	// CTLogPublicKeysPEM are the CT log public keys (PEM-encoded).
	// If empty, uses Sigstore public good instance.
	CTLogPublicKeysPEM []string
}

// TrustRootProvider provides trust root data for Sigstore verification.
type TrustRootProvider struct {
	config    *TrustRootConfig
	trustRoot root.TrustedMaterial
}

// NewPublicGoodTrustRoot creates a trust root provider using Sigstore public good instance.
// This fetches the latest trust root from Sigstore's TUF repository.
func NewPublicGoodTrustRoot() (*TrustRootProvider, error) {
	// Use sigstore-go's LiveTrustedRoot which automatically refreshes
	opts := tuf.DefaultOptions()

	liveTrustedRoot, err := root.NewLiveTrustedRoot(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create live trusted root: %w", err)
	}

	return &TrustRootProvider{
		config:    nil,
		trustRoot: liveTrustedRoot,
	}, nil
}

// NewStagingTrustRoot creates a trust root provider using Sigstore staging instance.
// This is used when signatures are created using the staging environment.
func NewStagingTrustRoot() (*TrustRootProvider, error) {
	opts := tuf.DefaultOptions()
	opts.RepositoryBaseURL = SigstoreStagingBaseURL

	liveTrustedRoot, err := root.NewLiveTrustedRoot(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create staging trusted root: %w", err)
	}

	return &TrustRootProvider{
		config:    nil,
		trustRoot: liveTrustedRoot,
	}, nil
}

// NewCustomTrustRoot creates a trust root provider using custom certificates and keys.
func NewCustomTrustRoot(config *TrustRootConfig) (*TrustRootProvider, error) {
	if config == nil {
		return nil, errors.New("config is required for custom trust root")
	}

	// If all fields are empty, fall back to public good
	if config.FulcioRootPEM == "" &&
		config.RekorPublicKeyPEM == "" &&
		len(config.TimestampAuthorityRootsPEM) == 0 &&
		len(config.CTLogPublicKeysPEM) == 0 {
		return NewPublicGoodTrustRoot()
	}

	// Build custom trusted root
	// For now, we create a simple implementation that validates against provided certs
	provider := &TrustRootProvider{
		config: config,
	}

	// Parse and validate the provided certificates/keys
	if config.FulcioRootPEM != "" {
		if _, err := parseCertificatesPEM(config.FulcioRootPEM); err != nil {
			return nil, fmt.Errorf("invalid Fulcio root certificate: %w", err)
		}
	}

	if config.RekorPublicKeyPEM != "" {
		if _, err := cryptoutils.UnmarshalPEMToPublicKey([]byte(config.RekorPublicKeyPEM)); err != nil {
			return nil, fmt.Errorf("invalid Rekor public key: %w", err)
		}
	}

	for i, tsaPEM := range config.TimestampAuthorityRootsPEM {
		if _, err := parseCertificatesPEM(tsaPEM); err != nil {
			return nil, fmt.Errorf("invalid TSA root certificate at index %d: %w", i, err)
		}
	}

	for i, ctLogPEM := range config.CTLogPublicKeysPEM {
		if _, err := cryptoutils.UnmarshalPEMToPublicKey([]byte(ctLogPEM)); err != nil {
			return nil, fmt.Errorf("invalid CT log public key at index %d: %w", i, err)
		}
	}

	return provider, nil
}

// GetTrustedMaterial returns the trusted material for verification.
func (p *TrustRootProvider) GetTrustedMaterial() (root.TrustedMaterial, error) {
	if p.trustRoot != nil {
		return p.trustRoot, nil
	}

	// For custom trust roots, we need to build the trusted material
	// This is a simplified implementation - full implementation would
	// construct a complete TrustedRoot from the config
	return nil, errors.New("custom trust root construction not fully implemented - use NewPublicGoodTrustRoot for now")
}

// GetFulcioCertificates returns the Fulcio CA certificates for verification.
func (p *TrustRootProvider) GetFulcioCertificates() ([]*x509.Certificate, error) {
	if p.config != nil && p.config.FulcioRootPEM != "" {
		return parseCertificatesPEM(p.config.FulcioRootPEM)
	}

	// Get from trusted material
	if p.trustRoot != nil {
		cas := p.trustRoot.FulcioCertificateAuthorities()

		var certs []*x509.Certificate
		for _, ca := range cas {
			if fca, ok := ca.(*root.FulcioCertificateAuthority); ok {
				if fca.Root != nil {
					certs = append(certs, fca.Root)
				}

				certs = append(certs, fca.Intermediates...)
			}
		}

		return certs, nil
	}

	return nil, errors.New("no Fulcio certificates available")
}

// IsCustom returns true if this is a custom trust root (not public good).
func (p *TrustRootProvider) IsCustom() bool {
	return p.config != nil
}

// parseCertificatesPEM parses PEM-encoded certificates.
func parseCertificatesPEM(pemData string) ([]*x509.Certificate, error) {
	certs, err := cryptoutils.UnmarshalCertificatesFromPEM([]byte(pemData))
	if err != nil {
		return nil, err
	}

	if len(certs) == 0 {
		return nil, errors.New("no certificates found in PEM data")
	}

	return certs, nil
}
