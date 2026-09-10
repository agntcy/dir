// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package install

import (
	cliconfig "github.com/agntcy/dir/cli/config"
	clientconfig "github.com/agntcy/dir/client/config"
)

// ActiveContextName returns the name of the client context this invocation
// talks to, following the same precedence dirctl itself uses: the --context
// flag, then DIRECTORY_CLIENT_CONTEXT, then current_context in the config file.
//
// It is recorded on every install and compared on every version check, because
// "is there a newer version?" is a question about one Directory. A package
// pulled from a colleague's Directory says nothing about the one configured
// now, and checking it against that one would report nonsense.
//
// An unresolvable context is the empty string, not an error. Validation is
// skipped and unknown fields tolerated, so a partially-set or forward-compat
// config still yields the name. An empty value means "no claim", and a row
// carrying one is checked against whatever context is active.
func ActiveContextName() string {
	_, resolved, err := clientconfig.Resolve(clientconfig.ResolveOptions{
		Context:            cliconfig.Context,
		SkipValidation:     true,
		AllowUnknownFields: true,
	})
	if err != nil || resolved == nil {
		return ""
	}

	return resolved.Name
}
