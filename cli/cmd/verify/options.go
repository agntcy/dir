// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:wrapcheck
package verify

import (
	signv1 "github.com/agntcy/dir/api/sign/v1"
	"github.com/agntcy/dir/cli/presenter"
)

var opts Options

type Options struct {
	// Key verification options
	Key string

	// OIDC verification options
	OIDCIssuer      string
	OIDCSubject     string
	TufMirrorUrl    string
	TrustedRootPath string
	IgnoreTlog      bool
	IgnoreTsa       bool
	IgnoreSct       bool

	// Output file flag
	OutputFile string
}

func init() {
	// Add output format flags
	presenter.AddOutputFlags(Command)

	// Add verification option flags
	Command.Flags().StringVar(&opts.Key, "key", "",
		`Public key to verify against. Accepts PEM content, file path, URL, or KMS URI.
Supported formats:
  - File path: /path/to/cosign.pub
  - HTTP(S) URL: https://example.com/cosign.pub
  - Environment variable: env://COSIGN_PUBLIC_KEY
  - AWS KMS: awskms://[ENDPOINT]/[ID/ALIAS/ARN]
  - GCP KMS: gcpkms://projects/[PROJECT]/locations/[LOC]/keyRings/[RING]/cryptoKeys/[KEY]
  - Azure Key Vault: azurekms://[VAULT_NAME][VAULT_URI]/[KEY]
  - Hashicorp Vault: hashivault://[KEY]
  - Kubernetes secret: k8s://[NAMESPACE]/[SECRET_NAME]
  - PKCS11 token: pkcs11:token=...;slot-id=...;object=...
  - GitLab: gitlab://[PROJECT]`)
	Command.Flags().StringVar(&opts.OIDCIssuer, "oidc-issuer", "", "OIDC issuer URL to verify against (e.g., https://github.com/login/oauth)")
	Command.Flags().StringVar(&opts.OIDCSubject, "oidc-subject", "", "OIDC subject/identity to verify against (e.g., user@example.com)")
	Command.Flags().StringVar(&opts.TufMirrorUrl, "tuf-mirror-url", signv1.DefaultVerifyOptionsOIDC.GetTufMirrorUrl(),
		"TUF repository mirror URL for fetching trusted root material")
	Command.Flags().StringVar(&opts.TrustedRootPath, "trusted-root-path", signv1.DefaultVerifyOptionsOIDC.GetTrustedRootPath(),
		"Path to a Sigstore TrustedRoot JSON file for offline/air-gapped verification")
	Command.Flags().BoolVar(&opts.IgnoreTlog, "ignore-tlog", signv1.DefaultVerifyOptionsOIDC.GetIgnoreTlog(),
		"Skip transparency log (Rekor) verification")
	Command.Flags().BoolVar(&opts.IgnoreTsa, "ignore-tsa", signv1.DefaultVerifyOptionsOIDC.GetIgnoreTsa(),
		"Skip timestamp authority (TSA) verification")
	Command.Flags().BoolVar(&opts.IgnoreSct, "ignore-sct", signv1.DefaultVerifyOptionsOIDC.GetIgnoreSct(),
		"Skip Signed Certificate Timestamp (SCT) verification")

	// Output file flag
	Command.Flags().StringVar(&opts.OutputFile, "output-file", "",
		"Write JSON output to file instead of stdout")

	// Mark flags as mutually exclusive
	Command.MarkFlagsMutuallyExclusive("key", "oidc-issuer")
	Command.MarkFlagsMutuallyExclusive("key", "oidc-subject")
	Command.MarkFlagsMutuallyExclusive("key", "tuf-mirror-url")
	Command.MarkFlagsMutuallyExclusive("key", "trusted-root-path")
	Command.MarkFlagsMutuallyExclusive("key", "ignore-tlog")
	Command.MarkFlagsMutuallyExclusive("key", "ignore-tsa")
	Command.MarkFlagsMutuallyExclusive("key", "ignore-sct")
}
