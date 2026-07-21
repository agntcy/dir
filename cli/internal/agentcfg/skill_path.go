// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package agentcfg

import "errors"

// ErrNoScopePath is returned by a resolver when the agent has no config location
// for the requested scope (global or project), signalling the engine to skip that
// artifact with a note rather than fail.
var ErrNoScopePath = errors.New("no config location for this scope")

// ResolveSkillPath resolves the on-disk skill path for the given scope, for
// display. It never returns an error (display-only).
func ResolveSkillPath(target *SkillTarget, env Env, slug string, scope Scope) string {
	path, err := resolveSkillTargetPath(target, env, slug, scope)
	if err != nil {
		if errors.Is(err, ErrNoScopePath) {
			return "(no " + scope.String() + " location for this agent)"
		}

		return "(path error: " + err.Error() + ")"
	}

	return path
}
