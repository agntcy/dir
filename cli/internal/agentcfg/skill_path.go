// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package agentcfg

import "errors"

// ErrNoGlobalPath is returned by a SkillTarget.Path resolver when the agent has
// no global mechanism for the skill artifact, signalling the engine to fall back
// to the project (cwd) path.
var ErrNoGlobalPath = errors.New("no global skill path for this agent")

// ResolveSkillPath resolves the on-disk path for a skill target for display: the
// global path when available, otherwise the project (cwd) fallback. It annotates
// a project fallback and never returns an error (display-only).
func ResolveSkillPath(target *SkillTarget, env Env, slug string) string {
	path, usedProject, err := resolveSkillTargetPath(target, env, slug)
	if err != nil {
		if errors.Is(err, ErrNoGlobalPath) {
			return "(no global path; run inside a repo for a project rule)"
		}

		return "(path error: " + err.Error() + ")"
	}

	if usedProject {
		return path + " (project fallback)"
	}

	return path
}
