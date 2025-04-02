package options

import "github.com/spf13/cobra"

type PushOptions struct {
	*BaseOption
	FromStdIn bool
}

func NewPushOptions() *PushOptions {
	opts := &PushOptions{
		BaseOption: &BaseOption{},
	}

	opts.AddRegisterFns([]RegisterFn{
		func(cmd *cobra.Command) error {
			flags := cmd.Flags()
			flags.BoolVar(&opts.FromStdIn, "stdin", false, "Read from stdin")
			return nil
		},
	})

	return opts
}
