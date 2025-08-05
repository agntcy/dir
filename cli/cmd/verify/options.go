// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package verify

var opts = &options{}

type options struct {
	FromStdin bool

	// Verification options
	OIDC         bool
	OIDCIssuer   string
	OIDCIdentity string
	Key          string
}

func init() {
	flags := Command.Flags()

	// Verification options
	flags.BoolVar(&opts.OIDC, "oidc", false,
		"Use OIDC identity-based verification.")
	flags.StringVar(&opts.OIDCIssuer, "oidc-issuer", ".*",
		"OIDC Issuer to check against. Accepts regular expressions.")
	flags.StringVar(&opts.OIDCIdentity, "oidc-identity", ".*",
		"OIDC Identity to compare against. Accepts regular expressions.")
	flags.StringVar(&opts.Key, "key", "",
		"Path to the public key file to use for verification (e.g., a Cosign public key). Use this option to verify signatures created with a self-managed keypair instead of OIDC identity-based verification.")
}
