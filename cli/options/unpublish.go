package options

import "github.com/spf13/cobra"

type UnpublishOptions struct {
	*BaseOption

	Network bool
}

func NewUnpublishOptions(baseOption *BaseOption, cmd *cobra.Command) *UnpublishOptions {
	opts := &UnpublishOptions{
		BaseOption: baseOption,
	}

	opts.AddRegisterFns(func() error {
		flags := cmd.Flags()
		flags.BoolVar(&opts.Network, "network", false, "Unpublish data from the network")

		return nil
	})

	return opts
}
