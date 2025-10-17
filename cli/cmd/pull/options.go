// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package pull

import "github.com/agntcy/dir/cli/presenter"

var opts = &options{}

type options struct {
	PublicKey bool
	Signature bool
}

func init() {
	flags := Command.Flags()
	flags.BoolVar(&opts.PublicKey, "public-key", false, "Pull the public key for the record.")
	flags.BoolVar(&opts.Signature, "signature", false, "Pull the signature for the record.")

	// Add output format flags
	presenter.AddOutputFlags(Command)
}
