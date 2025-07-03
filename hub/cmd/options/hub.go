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

	hubAddressConfigPath = "hub.server-address"
	insecureFlagName     = "insecure"
	insecureConfigPath   = "hub.insecure"
)

type HubOptions struct {
	*BaseOption

	ServerAddress string
	Insecure      bool
}

func NewHubOptions(base *BaseOption, cmd *cobra.Command) *HubOptions {
	hubOpts := &HubOptions{
		BaseOption: base,
	}

	hubOpts.AddRegisterFn(
		func() error {
			flags := cmd.PersistentFlags()
			flags.String(hubAddressFlagName, config.DefaultHubAddress, "AgentHub address")
			flags.Bool(insecureFlagName, false, "WARNING: Disables SSL certificate verification. This is insecure and not recommended for production use.")

			if err := viper.BindPFlag(hubAddressConfigPath, flags.Lookup(hubAddressFlagName)); err != nil {
				return fmt.Errorf("unable to bind flag %s: %w", hubAddressFlagName, err)
			}

			if err := viper.BindPFlag(insecureConfigPath, flags.Lookup(insecureFlagName)); err != nil {
				return fmt.Errorf("unable to bind flag %s: %w", insecureFlagName, err)
			}

			return nil
		},
	)

	hubOpts.AddCompleteFn(func() {
		hubOpts.ServerAddress = viper.GetString(hubAddressConfigPath)
		hubOpts.Insecure = viper.GetBool(insecureConfigPath)
	})

	return hubOpts
}
