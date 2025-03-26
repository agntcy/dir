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

	RootCmd.MarkFlagRequired("server-addr") //nolint:errcheck
}
