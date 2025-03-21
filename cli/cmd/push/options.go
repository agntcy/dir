// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package push

var opts = &options{}

type options struct {
	FromFile string
}

func init() {
	flags := Command.Flags()
	flags.StringVar(&opts.FromFile, "from-file", "", "Read compiled data from file, reads from STDIN if empty")
}
