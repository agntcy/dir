// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package init

var opts = &options{}

type options struct {
	Output string
}

func init() {
	flags := Command.Flags()
	flags.StringVarP(&opts.Output, "output", "o", "", "Path to the output file, where the generated private key will be stored.")
}
