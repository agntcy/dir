// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package pull

var opts = &options{}

type options struct {
	FormatRaw        bool
	IncludeSignature bool
}

func init() {
	flags := Command.Flags()
	flags.BoolVar(&opts.FormatRaw, "raw", false, "Output in Raw format. Defaults to JSON.")
	flags.BoolVar(&opts.IncludeSignature, "include-signature", false,
		"Include signature in the pull operation. "+
			"If true, the signature will be included in the pull operation.",
	)
}
