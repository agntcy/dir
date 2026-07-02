// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package install

// options holds the shared flags for the install subcommands.
type options struct {
	mcpOnly   bool
	skillOnly bool
	all       bool
	force     bool
	dryRun    bool
	yes       bool
}
