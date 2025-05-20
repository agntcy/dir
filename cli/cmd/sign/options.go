// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package sign

import (
	"github.com/agntcy/dir/client"
)

var opts = &options{}

type options struct {
	FromStdin bool

	// Signing options
	client.SignOpts
}

func init() {
	flags := Command.Flags()
	flags.BoolVar(&opts.FromStdin, "stdin", false,
		"Read data from standard input. Useful for piping. Reads from file if empty. "+
			"Ignored if file is provided as an argument.")

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
}
