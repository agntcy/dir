// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package cmd

var opts = &options{} //nolint:unused

//nolint:unused
type options struct {
	Query string
}

func init() {
	flags := RootCmd.PersistentFlags()
	flags.StringVar(&clientConfig.ServerAddress, "server-addr", clientConfig.ServerAddress, "Directory Server API address")
	flags.StringVar(&clientConfig.SpiffeSocketPath, "spiffe-socket-path", clientConfig.SpiffeSocketPath, "")
	flags.StringVar(&clientConfig.SpiffeTrustDomain, "spiffe-trust-domain", clientConfig.SpiffeTrustDomain, "")

	RootCmd.MarkFlagRequired("server-addr") //nolint:errcheck
}
