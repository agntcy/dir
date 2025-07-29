// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package push

var opts = &options{}

type options struct {
	FromStdin     bool
	SignaturePath string
}

func init() {
	flags := Command.Flags()
	flags.BoolVar(&opts.FromStdin, "stdin", false,
		"Read compiled data from standard input. Useful for piping. Reads from file if empty. "+
			"Ignored if file is provided as an argument.",
	)
	flags.StringVar(&opts.SignaturePath, "signature", "",
		"Path to signature file. If provided, the signature will be included in the push operation.",
	)
}
