// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package install

import "github.com/agntcy/dir/cli/cmd/search"

// options holds the shared flags for the install subcommands.
//
// Every field here must be bound to a flag. The local e2e suite runs dirctl
// in-process against this one package-level value, so a field nothing resets
// keeps whatever the previous spec left in it and turns test failures
// order-dependent.
type options struct {
	agents      []string
	project     bool
	dryRun      bool
	yes         bool
	pin         bool
	limit       uint32
	allVersions bool
	filters     search.Filters
}
