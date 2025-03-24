// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package publish

var opts = &options{}

type options struct {
	Local bool
}

func init() {
	flags := Command.Flags()
	flags.BoolVar(&opts.Local, "local", false, "Publish data only to local routing table")
}
