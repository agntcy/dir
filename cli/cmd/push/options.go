// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package push

import (
	"github.com/agntcy/dir/client"
)

var opts = &options{}

type options struct {
	FromStdin bool
	Sign      bool

	// Signing options
	client.SignOpts
}

func init() {
	flags := Command.Flags()
	flags.BoolVar(&opts.FromStdin, "stdin", false,
		"Read compiled data from standard input. Useful for piping. Reads from file if empty. "+
			"Ignored if file is provided as an argument.",
	)
	flags.BoolVar(&opts.Sign, "sign", false,
		"Sign the record with the specified signing options.",
	)

	// Signing options
	flags.StringVar(&opts.FulcioURL, "fulcio-url", client.DefaultFulcioURL,
		"Sigstore Fulcio URL")
	flags.StringVar(&opts.RekorURL, "rekor-url", client.DefaultRekorURL,
		"Sigstore Rekor URL")
	flags.StringVar(&opts.TimestampURL, "timestamp-url", client.DefaultTimestampURL,
		"Sigstore Timestamp URL")
	flags.StringVar(&opts.OIDCProviderURL, "oidc-provider-url", client.DefaultOIDCProviderURL,
		"OIDC Provider URL")
	flags.StringVar(&opts.OIDCClientID, "oidc-client-id", client.DefaultOIDCClientID,
		"OIDC Client ID")
	flags.StringVar(&opts.OIDCToken, "oidc-token", "",
		"OIDC Token for non-interactive signing. ")
	flags.StringVar(&opts.Key, "key", "",
		"Path to the private key file to use for signing (e.g., a Cosign key generated with a GitHub token). Use this option to sign with a self-managed keypair instead of OIDC identity-based signing.")
}
