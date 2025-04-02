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
		flags.BoolVar(&opts.FromStdIn, "stdin", false, "Read from stdin")
		return nil
	})

	return opts
}
