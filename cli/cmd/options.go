// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package cmd

var opts = &options{}

type options struct {
	Query string
}

func init() {
	flags := RootCmd.Flags()
	flags.StringVar(&clientConfig.ServerAddress, "server-addr", clientConfig.ServerAddress, "Directory Server API address")

	RootCmd.MarkFlagRequired("server-addr")
}
