// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package sync

var opts = &options{}

type options struct {
	Limit  uint32
	Offset uint32
}

//nolint:mnd
func init() {
	flags := listCmd.Flags()

	flags.Uint32Var(&opts.Limit, "limit", 100, "Maximum number of sync operations to return (default: 100)")
	flags.Uint32Var(&opts.Offset, "offset", 0, "Number of sync operations to skip (for pagination)")
}
