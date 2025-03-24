// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package peers

var opts = &options{}

type options struct {
	Digest string
}

func init() {
	flags := Command.Flags()
	flags.StringVar(&opts.Digest, "digest", "", "Digest to search for peers")

	Command.MarkFlagRequired("digest")
}
