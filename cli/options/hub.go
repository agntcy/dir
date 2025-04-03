package options

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/agntcy/dir/cli/config"
)

const (
	hubAddressFlagName   = "server-address"
	hubAddressConfigPath = "hub.server-address"
)

type HubOptions struct {
	*BaseOption

	ServerAddress string
}

func NewHubOptions(base *BaseOption, cmd *cobra.Command) *HubOptions {
	hubOpts := &HubOptions{
		BaseOption: base,
	}

	hubOpts.AddRegisterFns(
		func() error {
			flags := cmd.PersistentFlags()
			flags.String(hubAddressFlagName, config.DefaultHubAddress, "AgentHub address")
			if err := viper.BindPFlag(hubAddressConfigPath, flags.Lookup(hubAddressFlagName)); err != nil {
				return err
			}
			hubOpts.ServerAddress = viper.GetString(hubAddressConfigPath)
			return nil
		},
	)

	return hubOpts
}
