// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package v1

const (
	DefaultFulcioURL       = "https://fulcio.sigstore.dev"
	DefaultRekorURL        = "https://rekor.sigstore.dev"
	DefaultTimestampURL    = "https://timestamp.sigstore.dev/api/v1/timestamp"
	DefaultTUFMirrorURL    = "https://tuf-repo-cdn.sigstore.dev"
	DefaultOIDCProviderURL = "https://oauth2.sigstore.dev/auth"
	DefaultOIDCClientID    = "sigstore"
)

var (
	// DefaultSignOptionsOIDC provides default values for OIDC-based signing.
	DefaultSignOptionsOIDC = &SignOptionsOIDC{
		FulcioUrl:        DefaultFulcioURL,
		RekorUrl:         DefaultRekorURL,
		TimestampUrl:     DefaultTimestampURL,
		SkipTlog:         false,
		OidcProviderUrl:  DefaultOIDCProviderURL,
		OidcClientId:     DefaultOIDCClientID,
		OidcClientSecret: "",
	}

	// DefaultVerifyOptionsOIDC provides default values for OIDC-based verification.
	DefaultVerifyOptionsOIDC = &VerifyOptionsOIDC{
		TufMirrorUrl:    DefaultTUFMirrorURL,
		TrustedRootPath: "",
		IgnoreTlog:      false,
		IgnoreTsa:       false,
		IgnoreSct:       false,
	}
)

// GetDefaultOptions returns SignOptionsOIDC with defaults applied for empty fields.
func (x *SignOptionsOIDC) GetDefaultOptions() *SignOptionsOIDC {
	if x == nil {
		return DefaultSignOptionsOIDC
	}

	return &SignOptionsOIDC{
		FulcioUrl:        valueOrDefault(x.GetFulcioUrl(), DefaultSignOptionsOIDC.GetFulcioUrl()),
		RekorUrl:         valueOrDefault(x.GetRekorUrl(), DefaultSignOptionsOIDC.GetRekorUrl()),
		TimestampUrl:     valueOrDefault(x.GetTimestampUrl(), DefaultSignOptionsOIDC.GetTimestampUrl()),
		SkipTlog:         valueOrDefault(x.GetSkipTlog(), DefaultSignOptionsOIDC.GetSkipTlog()),
		OidcProviderUrl:  valueOrDefault(x.GetOidcProviderUrl(), DefaultSignOptionsOIDC.GetOidcProviderUrl()),
		OidcClientId:     valueOrDefault(x.GetOidcClientId(), DefaultSignOptionsOIDC.GetOidcClientId()),
		OidcClientSecret: x.GetOidcClientSecret(), // No default, keep user value
	}
}

// GetDefaultOptions returns VerifyOptionsOIDC with defaults applied for empty fields.
func (x *VerifyOptionsOIDC) GetDefaultOptions() *VerifyOptionsOIDC {
	if x == nil {
		return DefaultVerifyOptionsOIDC
	}

	return &VerifyOptionsOIDC{
		TufMirrorUrl:    valueOrDefault(x.GetTufMirrorUrl(), DefaultVerifyOptionsOIDC.GetTufMirrorUrl()),
		TrustedRootPath: x.GetTrustedRootPath(), // No default, keep user value
		IgnoreTlog:      valueOrDefault(x.GetIgnoreTlog(), DefaultVerifyOptionsOIDC.GetIgnoreTlog()),
		IgnoreTsa:       valueOrDefault(x.GetIgnoreTsa(), DefaultVerifyOptionsOIDC.GetIgnoreTsa()),
		IgnoreSct:       valueOrDefault(x.GetIgnoreSct(), DefaultVerifyOptionsOIDC.GetIgnoreSct()),
	}
}

// valueOrDefault returns the value if it is not the zero value, otherwise returns the defaultValue.
func valueOrDefault[T comparable](value, defaultValue T) T {
	var zero T
	if value != zero {
		return value
	}

	return defaultValue
}
