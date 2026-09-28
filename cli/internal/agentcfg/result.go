// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package agentcfg

// The artifact kinds reported in Outcome.Artifact.
const (
	// ArtifactMCP is an MCP server entry in an agent's config file.
	ArtifactMCP = "mcp"
	// ArtifactSkill is an Agent Skill file, folder, or managed block.
	ArtifactSkill = "skill"
)

// Action is the outcome of touching a single artifact location.
type Action string

const (
	// ActionAdded means a new entry/file was created.
	ActionAdded Action = "added"
	// ActionUpdated means an existing entry/file of ours was replaced.
	ActionUpdated Action = "updated"
	// ActionRemoved means our entry/file was deleted.
	ActionRemoved Action = "removed"
	// ActionUnchanged means our entry/file already matched; nothing was written.
	ActionUnchanged Action = "unchanged"
	// ActionSkipped means we deliberately did not act (with a reason).
	ActionSkipped Action = "skipped"
	// ActionFailed means an error occurred for this artifact.
	ActionFailed Action = "failed"
)

// Changed reports whether the action altered anything on disk.
func (a Action) Changed() bool {
	switch a {
	case ActionAdded, ActionUpdated, ActionRemoved:
		return true
	case ActionUnchanged, ActionSkipped, ActionFailed:
		return false
	default:
		return false
	}
}

// HasChanges reports whether applying these outcomes would alter anything.
//
// A plan in which nothing changes is a plan not worth confirming, so callers
// use this to stop before the prompt rather than ask the user to approve a
// no-op.
func HasChanges(outcomes []Outcome) bool {
	for _, o := range outcomes {
		if o.Action.Changed() {
			return true
		}
	}

	return false
}

// reportable drops the outcomes a reader does not need to see.
//
// An unchanged outcome is one where nothing was written and nothing will be:
// the artifact was already correct, or — far more often — the agent never had
// it in the first place. Listing those alongside the real changes reads as a
// claim that the package is installed in every agent named, which is exactly
// backwards. Skips and failures stay, because each one explains why an agent
// the user asked for got nothing.
//
// The tally still counts every outcome, so the unchanged ones are not hidden,
// only kept out of the per-agent lines.
func reportable(outcomes []Outcome) []Outcome {
	var shown []Outcome

	for _, o := range outcomes {
		if o.Action != ActionUnchanged {
			shown = append(shown, o)
		}
	}

	return shown
}

// failOutcome marks an outcome as failed with err and returns both, so callers
// can `return failOutcome(outcome, fmt.Errorf(...))` in one line.
func failOutcome(outcome Outcome, err error) (Outcome, error) {
	outcome.Action = ActionFailed
	outcome.Err = err

	return outcome, err
}

// skipScopeOutcome builds a skipped outcome for an artifact that has no config
// location for the requested scope, so a missing project/global path degrades
// gracefully instead of failing the run.
func skipScopeOutcome(artifact string, scope Scope) Outcome {
	return Outcome{
		Artifact: artifact,
		Action:   ActionSkipped,
		Reason:   "no " + scope.String() + " location for this agent's " + artifact,
	}
}

// Outcome records what happened to one artifact (MCP or skill) for one agent.
type Outcome struct {
	Record   string // optional record label for batch install grouping
	Agent    string // human-readable agent name
	Artifact string // ArtifactMCP or ArtifactSkill
	Path     string // absolute path that was (or would be) touched
	Action   Action
	Reason   string // populated on skip/fail or notable fallbacks
	Err      error

	// Server is the config key an MCP server entry was stored under, empty for
	// skills. One agent gets one outcome per server, all sharing a Path, so this
	// is the only thing that tells them apart — which is what lets the install
	// manifest record the server keys that were actually written.
	Server string
}
