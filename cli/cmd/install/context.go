// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package install

import (
	"strings"

	cliconfig "github.com/agntcy/dir/cli/config"
)

// ActiveDirectory returns the server address this invocation talks to.
//
// It is recorded on every install and compared on every version check, because
// "is there a newer version?" is a question about one Directory. A package
// pulled from a colleague's Directory says nothing about the one configured
// now, and checking it against that one would report nonsense.
//
// The address rather than the context name, because a name is not an identity.
// `DIRECTORY_CLIENT_SERVER_ADDRESS` and `--server-addr` both replace a
// context's endpoint while leaving its name in place, so an install made
// against an overridden endpoint would be recorded under the context it did
// not actually use, and later compared against that context's original server.
// Two contexts pointing at one Directory are also the same Directory, which
// the address gets right and the name does not.
//
// This reads the config that root.go already resolved, so every override the
// invocation applied is included. An address that could not be resolved is the
// empty string, which means "no claim": such a row is checked against whatever
// Directory is active rather than skipped forever.
func ActiveDirectory() string {
	if cliconfig.Client == nil {
		return ""
	}

	return strings.TrimSpace(cliconfig.Client.ServerAddress)
}
