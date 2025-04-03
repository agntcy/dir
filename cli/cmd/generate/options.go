// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package generate

var opts = &options{} //nolint:unused

//nolint:unused
type options struct {
	PrivateKeyFilePath string
}

func init() {
	flags := Command.PersistentFlags()
	flags.StringVarP(&opts.PrivateKeyFilePath, "private-key-file-path", "p", "", "The path to the private key on the filesystem. This key will be used to generate the peer id.")

	Command.MarkFlagRequired("private-key-file-path")
}
