// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package sign

import (
	signv1 "github.com/agntcy/dir/api/sign/v1"
	"github.com/agntcy/dir/cli/presenter"
	"github.com/spf13/pflag"
)

var opts = &Options{}

type Options struct {
	// Key signing options
	Key string

	// OIDC signing options
	FulcioURL        string
	RekorURL         string
	TimestampURL     string
	SkipTlog         bool
	OIDCProviderURL  string
	OIDCClientID     string
	OIDCClientSecret string
	OIDCToken        string
}

func AddSigningFlags(flags *pflag.FlagSet) {
	flags.StringVar(&opts.Key, "key", "",
		`Private key for signing. Accepts PEM content, file path, URL, or KMS URI.
Supported formats:
  - File path: /path/to/cosign.key
  - HTTP(S) URL: https://example.com/cosign.key
  - Environment variable: env://COSIGN_PRIVATE_KEY
  - AWS KMS: awskms://[ENDPOINT]/[ID/ALIAS/ARN]
  - GCP KMS: gcpkms://projects/[PROJECT]/locations/[LOC]/keyRings/[RING]/cryptoKeys/[KEY]
  - Azure Key Vault: azurekms://[VAULT_NAME][VAULT_URI]/[KEY]
  - Hashicorp Vault: hashivault://[KEY]
  - Kubernetes secret: k8s://[NAMESPACE]/[SECRET_NAME]
  - PKCS11 token: pkcs11:token=...;slot-id=...;object=...
  - GitLab: gitlab://[PROJECT]`)
	flags.StringVar(&opts.FulcioURL, "fulcio-url", signv1.DefaultFulcioURL,
		"Sigstore Fulcio URL")
	flags.StringVar(&opts.RekorURL, "rekor-url", signv1.DefaultRekorURL,
		"Sigstore Rekor URL")
	flags.StringVar(&opts.TimestampURL, "timestamp-url", signv1.DefaultTimestampURL,
		"Sigstore Timestamp URL")
	flags.BoolVar(&opts.SkipTlog, "skip-tlog", false,
		"Skip uploading signature to transparency log (Rekor)")
	flags.StringVar(&opts.OIDCProviderURL, "oidc-provider-url", signv1.DefaultOIDCProviderURL,
		"OIDC Provider URL")
	flags.StringVar(&opts.OIDCClientID, "oidc-client-id", signv1.DefaultOIDCClientID,
		"OIDC Client ID")
	flags.StringVar(&opts.OIDCClientSecret, "oidc-client-secret", "",
		"OIDC Client Secret (required for confidential OIDC clients)")
	flags.StringVar(&opts.OIDCToken, "oidc-token", "",
		"OIDC Token for non-interactive signing")
}

func init() {
	flags := Command.Flags()

	// Add signing flags
	AddSigningFlags(flags)

	// Add output format flags
	presenter.AddOutputFlags(Command)
}
