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

func NewOptions() *HubOptions {
	return &HubOptions{
		BaseOption: &BaseOption{},
	}
}
func (o *HubOptions) Register(cmd *cobra.Command) {
	// server address
	flags := cmd.PersistentFlags()
	flags.String(hubAddressFlagName, config.DefaultHubAddress, "Address of the Phoenix SaaS hub server")
	if err := viper.BindPFlag(hubAddressConfigPath, flags.Lookup(hubAddressFlagName)); err != nil {
		o.AddError(err)
	}
}

func (o *HubOptions) Complete() {
	o.ServerAddress = viper.GetString(hubAddressConfigPath)
}
