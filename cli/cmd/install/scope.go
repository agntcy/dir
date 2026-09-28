// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package install

import (
	"os"
	"strings"

	"github.com/agntcy/dir/cli/internal/agentcfg"
	"github.com/agntcy/dir/cli/internal/pkgstate"
)

// manifestScope names where a row's artifacts landed. A project install
// records the repository it wrote into rather than the word "project", so one
// manifest can hold installs from every repository without two of them
// colliding on the same (name, agent, scope) key.
func manifestScope(scope agentcfg.Scope) pkgstate.Scope {
	if scope == agentcfg.Global {
		return pkgstate.ScopeGlobal
	}

	return pkgstate.ProjectScope(agentcfg.ResolveEnv().Cwd)
}

// shortScope renders a scope for the human views, abbreviating the user's home
// directory to `~` so a repository path does not push a table off the screen.
// Machine-readable output keeps the path exactly as recorded, since a consumer
// has to be able to use it.
func shortScope(scope pkgstate.Scope) string {
	dir := scope.Dir()
	if dir == "" {
		return string(scope)
	}

	home, err := os.UserHomeDir()
	if err != nil || home == "" || !strings.HasPrefix(dir, home+string(os.PathSeparator)) {
		return dir
	}

	return "~" + dir[len(home):]
}
