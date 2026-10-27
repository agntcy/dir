// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package policy

import (
	"context"

	policyv1 "github.com/agntcy/dir/api/policy/v1"
	"github.com/agntcy/dir/server/authn"
	policyconfig "github.com/agntcy/dir/server/policy/config"
	"github.com/agntcy/dir/utils/logging"
)

// logger records every operation decision.
//
// An operation policy persists nothing, so this log is the only trace a
// decision leaves. The Prometheus counters answer a different question: they
// aggregate, and cannot say which record was refused by which policy.
var logger = logging.Logger("policy")

// What a decision was, as one word, so a line can be selected on it without
// matching the message text.
const (
	decisionAllow  = "allow"
	decisionRefuse = "refuse"
)

// logDecision writes the one line that records a decision.
//
// A shadow policy's decision is dropped by Decide and never reaches the
// caller, so this is the whole of what shadow mode produces: the log is the
// mechanism rather than an addition to it. Which is why the line carries the
// real decision and reports the mode beside it, instead of reporting a shadow
// refusal as something that did not happen.
//
// The record itself is never logged, only its CID. The record is caller
// supplied content and the context carries authenticated identity; neither
// belongs in an access log.
func logDecision(ctx context.Context, p operationPolicy, input *policyv1.OperationInput, allowed bool, msgs []string) {
	decision := decisionRefuse
	if allowed {
		decision = decisionAllow
	}

	opCtx := input.GetContext()

	attrs := []any{
		"operation", opCtx.GetOperation().String(),
		"cid", input.GetRecordCid(),
		"policy_id", p.id,
		"mode", string(p.mode),
		"decision", decision,
	}

	// Who asked, where it is known. A record pulled by autosync has a peer and
	// no caller; a pushed one has a caller and no peer. Absent either way
	// rather than empty, so a line never names a peer or caller of "".
	if peerID := opCtx.GetSourcePeerId(); peerID != "" {
		attrs = append(attrs, "peer", peerID)
	}

	if id, ok := authn.SpiffeIDFromContext(ctx); ok {
		attrs = append(attrs, "caller", id.String())
	}

	if allowed {
		// Every push, autosync and index of every record passes through here,
		// so an allowed decision is only worth a line when asked for.
		logger.Debug("Operation policy allowed the record", attrs...)

		return
	}

	// The messages are the policy's own, and are already returned to the
	// caller when the policy enforces.
	attrs = append(attrs, "reason", msgs)

	if p.mode == policyconfig.ModeShadow {
		logger.Info("Shadow mode: operation policy would have refused the record", attrs...)

		return
	}

	logger.Info("Operation policy refused the record", attrs...)
}
