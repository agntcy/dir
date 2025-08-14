// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package pull

var opts = &options{}

type options struct {
	FormatRaw bool
}

func init() {
	flags := Command.Flags()
	flags.BoolVar(&opts.FormatRaw, "raw", false, "Output in Raw format. Defaults to JSON.")
}
