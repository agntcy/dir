// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package policy

import (
	"fmt"

	policyv1 "github.com/agntcy/dir/api/policy/v1"
	"github.com/agntcy/dir/server/types"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// VerdictSource is what the evaluation policies concluded about a record.
//
// An interface rather than the database itself so that this package keeps no
// opinion about where verdicts are stored: the server satisfies it with the
// database it already has, and a test with whatever it likes.
type VerdictSource interface {
	// GetPolicyEvaluations returns every verdict recorded for a record, at
	// most one per policy.
	GetPolicyEvaluations(recordCID string) ([]types.PolicyEvaluationObject, error)
}

// EnforcementSource is which version of each policy is currently in force,
// and which policies have finished covering the catalogue.
//
// Read per decision rather than held, because it changes underneath: a policy
// finishes backfilling while the server runs, and a decision made after that
// has to see it.
type EnforcementSource func() types.PolicyEnforcement

// verdictsFor builds the map an operation policy reads, keyed by every
// configured evaluation policy.
//
// Every configured policy gets an entry, including the ones this record has
// no row for. That is what makes VERDICT_STATUS_UNEVALUATED — "configured,
// not evaluated yet" — distinguishable from a key that names no policy at
// all, which is a mistake in the expression rather than a state of the
// record.
//
// Rows for policies that are no longer configured are dropped. A verdict
// under a policy nobody asked for is not an answer to any question being
// decided now.
func verdictsFor(ids []string, rows []types.PolicyEvaluationObject, inForce map[string]string) map[string]*policyv1.Verdict {
	byID := make(map[string]types.PolicyEvaluationObject, len(rows))
	for _, row := range rows {
		byID[row.GetPolicyID()] = row
	}

	verdicts := make(map[string]*policyv1.Verdict, len(ids))

	for _, id := range ids {
		row, found := byID[id]
		if !found {
			verdicts[id] = &policyv1.Verdict{Status: policyv1.VerdictStatus_VERDICT_STATUS_UNEVALUATED}

			continue
		}

		verdicts[id] = verdictOf(row, inForce[id])
	}

	return verdicts
}

// verdictOf reads one stored row as the verdict a policy sees, given the
// version of its policy in force.
//
// compliant_and_current is derived here and nowhere else. It is the
// fail-closed property the gate rests on, and every way of being stale —
// no verdict, a failed one, one whose metadata moved, one reached under a
// version since edited — has to arrive at the same false.
func verdictOf(row types.PolicyEvaluationObject, versionInForce string) *policyv1.Verdict {
	status := verdictStatusOf(row, versionInForce)
	compliant := row.GetCompliant() && status == policyv1.VerdictStatus_VERDICT_STATUS_EVALUATED

	verdict := &policyv1.Verdict{
		Status:              status,
		Compliant:           compliant,
		CompliantAndCurrent: compliant,
		PolicyVersion:       row.GetPolicyVersion(),
		Reason:              row.GetReason(),
	}

	if at := row.GetUpdatedAt(); !at.IsZero() {
		verdict.EvaluatedAt = timestamppb.New(at)
	}

	return verdict
}

// verdictStatusOf maps a stored status onto what a policy is told.
//
// The stored statuses and the reported ones are not the same vocabulary, and
// the difference is the version. A row stored as evaluated is only reported
// that way while the version it was reached under is the one in force:
// afterwards it is a conclusion about a different rule, which is what the
// reported superseded means. The stored superseded is a different event — the
// metadata underneath the verdict moved — and reports the same way because a
// policy can do nothing different about either.
//
// An empty version in force means no evaluator has registered the policy, so
// nothing a row says about it is current.
func verdictStatusOf(row types.PolicyEvaluationObject, versionInForce string) policyv1.VerdictStatus {
	switch row.GetStatus() {
	case types.PolicyEvalStatusEvaluated:
		if versionInForce == "" || row.GetPolicyVersion() != versionInForce {
			return policyv1.VerdictStatus_VERDICT_STATUS_SUPERSEDED
		}

		return policyv1.VerdictStatus_VERDICT_STATUS_EVALUATED
	case types.PolicyEvalStatusFailed:
		return policyv1.VerdictStatus_VERDICT_STATUS_FAILED
	case types.PolicyEvalStatusSuperseded:
		return policyv1.VerdictStatus_VERDICT_STATUS_SUPERSEDED
	default:
		// A status this build does not know. Reporting it as unspecified is
		// the one answer that cannot be mistaken for a conclusion.
		return policyv1.VerdictStatus_VERDICT_STATUS_UNSPECIFIED
	}
}

// versionsInForce is the version of each policy currently enforced or
// pending, by ID.
//
// Pending policies are included because their version is just as real: the
// latch decides whether a policy may be acted on, not whether the version it
// would be acted on under is known.
func versionsInForce(enforcement types.PolicyEnforcement) map[string]string {
	versions := make(map[string]string, len(enforcement.Policies)+len(enforcement.Pending))

	for _, p := range enforcement.Policies {
		versions[p.ID] = p.Version
	}

	for _, p := range enforcement.Pending {
		versions[p.ID] = p.Version
	}

	return versions
}

// readVerdicts collects the verdicts for one record.
func (r *Registry) readVerdicts(recordCID string) (map[string]*policyv1.Verdict, error) {
	rows, err := r.verdicts.GetPolicyEvaluations(recordCID)
	if err != nil {
		return nil, fmt.Errorf("read verdicts for %s: %w", recordCID, err)
	}

	return verdictsFor(r.evaluationIDs(), rows, versionsInForce(r.enforcement())), nil
}

// evaluationIDs are the configured evaluation policies, in config order.
func (r *Registry) evaluationIDs() []string {
	ids := make([]string, 0, len(r.evaluations))
	for _, e := range r.evaluations {
		ids = append(ids, e.id)
	}

	return ids
}

// enforcedIDs are the evaluation policies whose verdicts cover the catalogue,
// which is to say the ones a decision may be acted on.
func (r *Registry) enforcedIDs() map[string]struct{} {
	enforcement := r.enforcement()

	ids := make(map[string]struct{}, len(enforcement.Policies))
	for _, p := range enforcement.Policies {
		ids[p.ID] = struct{}{}
	}

	return ids
}

// mayEnforce reports whether a policy reading refs can be acted on yet.
//
// This is the backfill latch, which the SQL gate gets for free by leaving an
// uncovered policy out of its query. An expression cannot drop a term, so the
// latch has to hold the whole policy to reporting instead: requiring a
// verdict that does not exist for most of the catalogue would hide most of
// the catalogue, which is exactly what the latch exists to prevent.
//
// A policy reading no verdict is never held: there is nothing for it to wait
// on. A reference whose keys could not be read waits for every configured
// evaluation policy, because any of them might be the one meant.
func (r *Registry) mayEnforce(refs verdictRefs) bool {
	if !refs.reads() {
		return true
	}

	enforced := r.enforcedIDs()

	keys := refs.keys
	if refs.all {
		keys = r.evaluationIDs()
	}

	for _, key := range keys {
		if _, ok := enforced[key]; !ok {
			return false
		}
	}

	return true
}
