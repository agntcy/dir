// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package sign

import (
	"github.com/agntcy/dir/cli/presenter"
	"github.com/agntcy/dir/client"
	"github.com/agntcy/dir/utils/cosign"
	"github.com/spf13/pflag"
)

var opts = &options{}

type options struct {
	// Signing options
	client.SignOpts
}

func init() {
	flags := Command.Flags()

	AddSigningFlags(flags)

	// Add output format flags
	presenter.AddOutputFlags(Command)
}

func AddSigningFlags(flags *pflag.FlagSet) {
	flags.StringVar(&opts.FulcioURL, "fulcio-url", cosign.DefaultFulcioURL,
		"Sigstore Fulcio URL")
	flags.StringVar(&opts.RekorURL, "rekor-url", cosign.DefaultRekorURL,
		"Sigstore Rekor URL")
	flags.StringVar(&opts.TimestampURL, "timestamp-url", cosign.DefaultTimestampURL,
		"Sigstore Timestamp URL")
	flags.StringVar(&opts.OIDCProviderURL, "oidc-provider-url", cosign.DefaultOIDCProviderURL,
		"OIDC Provider URL")
	flags.StringVar(&opts.OIDCClientID, "oidc-client-id", cosign.DefaultOIDCClientID,
		"OIDC Client ID")
	flags.StringVar(&opts.OIDCToken, "oidc-token", "",
		"OIDC Token for non-interactive signing. ")
	flags.StringVar(&opts.Key, "key", "",
		"Path to the private key file to use for signing (e.g., a Cosign key generated with a GitHub token). Use this option to sign with a self-managed keypair instead of OIDC identity-based signing.")
}
