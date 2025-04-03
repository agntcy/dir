package options

import "github.com/spf13/cobra"

type PushOptions struct {
	*BaseOption
	FromStdIn bool
}

func NewPushOptions(base *BaseOption, cmd *cobra.Command) *PushOptions {
	opts := &PushOptions{
		BaseOption: base,
	}

	opts.AddRegisterFns(func() error {
		flags := cmd.Flags()
		flags.BoolVar(&opts.FromStdIn, "stdin", false,
			"Read compiled data from standard input. Useful for piping. Reads from file if empty. "+
				"Ignored if file is provided as an argument.",
		)
		return nil
	})

	return opts
}
