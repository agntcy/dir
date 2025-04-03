package options

import "github.com/spf13/cobra"

type ListInfoOptions struct {
	*BaseOption

	PeerID  string
	Network bool
}

func NewListInfoOptions(baseOption *BaseOption, cmd *cobra.Command) *ListInfoOptions {
	opts := &ListInfoOptions{
		BaseOption: baseOption,
	}

	opts.AddRegisterFns(func() error {
		flags := cmd.Flags()
		flags.StringVar(&opts.PeerID, "peer", "", "Get publication summary for a single peer")
		flags.BoolVar(&opts.Network, "network", false, "Get publication summary for the network")
		if err := flags.MarkHidden("peer"); err != nil {
			return err
		}
		cmd.MarkFlagsMutuallyExclusive("peer", "network")

		return nil
	})

	return opts
}
