// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package options

import (
	"fmt"

	"github.com/agntcy/dir/hub/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	hubAddressFlagName = "server-address"
	hubAPIKeyFileName  = "apikey-file"

	hubAddressConfigPath    = "hub.server-address"
	hubAPIKeyFileConfigPath = "hub.apikey-file" //nolint:gosec
)

type HubOptions struct {
	*BaseOption

	ServerAddress string
	APIKeyFile    string
}

func NewHubOptions(base *BaseOption, cmd *cobra.Command) *HubOptions {
	hubOpts := &HubOptions{
		BaseOption: base,
	}

	hubOpts.AddRegisterFn(
		func() error {
			flags := cmd.PersistentFlags()
			flags.String(hubAddressFlagName, config.DefaultHubAddress, "AgentHub address")
			flags.String(hubAPIKeyFileName, "", `Path to a JSON file containing API key credentials (format: {"client_id": "...", "secret": "..."})`)

			if err := viper.BindPFlag(hubAddressConfigPath, flags.Lookup(hubAddressFlagName)); err != nil {
				return fmt.Errorf("unable to bind flag %s: %w", hubAddressFlagName, err)
			}

			if err := viper.BindPFlag(hubAPIKeyFileConfigPath, flags.Lookup(hubAPIKeyFileName)); err != nil {
				return fmt.Errorf("unable to bind flag %s: %w", hubAPIKeyFileName, err)
			}

			return nil
		},
	)

	hubOpts.AddCompleteFn(func() {
		hubOpts.ServerAddress = viper.GetString(hubAddressConfigPath)
		hubOpts.APIKeyFile = viper.GetString(hubAPIKeyFileConfigPath)
	})

	return hubOpts
}
