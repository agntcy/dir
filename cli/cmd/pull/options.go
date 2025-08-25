// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package pull

var opts = &options{}

type options struct {
	FormatRaw bool
	PublicKey bool
	Signature bool
}

func init() {
	flags := Command.Flags()
	flags.BoolVar(&opts.FormatRaw, "raw", false, "Output in Raw format. Defaults to JSON.")
	flags.BoolVar(&opts.PublicKey, "public-key", false, "Pull the public key for the record.")
	flags.BoolVar(&opts.Signature, "signature", false, "Pull the signature for the record.")
}
