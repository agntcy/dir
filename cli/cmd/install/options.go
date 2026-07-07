// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package install

import "github.com/agntcy/dir/cli/cmd/search"

// options holds the shared flags for the install subcommands.
type options struct {
	agents      []string
	dryRun      bool
	yes         bool
	limit       uint32
	allVersions bool
	filters     search.Filters
}
