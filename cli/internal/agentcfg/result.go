// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package agentcfg

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
	Artifact string // "mcp" or "skill"
	Path     string // absolute path that was (or would be) touched
	Action   Action
	Reason   string // populated on skip/fail or notable fallbacks
	Err      error
}
