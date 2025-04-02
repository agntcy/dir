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

func NewHubOptions(bases ...*BaseOption) *HubOptions {
	base := &BaseOption{}
	if len(bases) > 0 {
		base = bases[0]
	}

	hubOpts := &HubOptions{
		BaseOption: base,
	}

	hubOpts.AddRegisterFns([]RegisterFn{
		func(cmd *cobra.Command) error {
			flags := cmd.Flags()
			flags.String(hubAddressFlagName, config.DefaultHubAddress, "AgentHub address")
			if err := viper.BindPFlag(hubAddressConfigPath, flags.Lookup(hubAddressFlagName)); err != nil {
				return err
			}
			return nil
		},
	})

	hubOpts.AddCompleteFn([]CompleteFn{
		func() {
			hubOpts.ServerAddress = viper.GetString(hubAddressConfigPath)
		},
	})

	return hubOpts
}
