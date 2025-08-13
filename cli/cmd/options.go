// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"github.com/agntcy/dir/client"
)

var (
	clientConfig = &client.DefaultConfig
)

func init() {
	// load config
	if cfg, err := client.LoadConfig(); err == nil {
		clientConfig = cfg
	}

	// set flags
	flags := RootCmd.PersistentFlags()
	flags.StringVar(&clientConfig.ServerAddress, "server-addr", clientConfig.ServerAddress, "Directory Server API address")
	flags.StringVar(&clientConfig.SpiffeSocketPath, "spiffe-socket-path", clientConfig.SpiffeSocketPath, "")
	flags.StringVar(&clientConfig.SpiffeTrustDomain, "spiffe-trust-domain", clientConfig.SpiffeTrustDomain, "")

	RootCmd.MarkFlagRequired("server-addr") //nolint:errcheck
}
