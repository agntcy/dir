// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package metrics

import (
	policyconfig "github.com/agntcy/dir/server/policy/config"
	"github.com/agntcy/dir/server/types"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	// outcomeExcluded labels what enforcing reads excluded.
	outcomeExcluded = "excluded"

	// outcomeWouldExclude labels what shadow-mode reads would have excluded.
	outcomeWouldExclude = "would_exclude"
)

// RecordCounter counts indexed records: those an enforcing search excludes
// under a set of policies, and those with no verdict yet under a policy's
// current version.
type RecordCounter interface {
	CountRecordsExcluded(policies []types.EnforcedPolicy, search []types.SearchPolicy) (int64, error)
	CountRecordsNeedingPolicyEvaluation(policyID, policyVersion string) (int64, error)
}

// PolicyGate reports what the content-policy gate excludes and, in shadow
// mode, what it would: the indexed records searches leave out, counted at
// each scrape, and the reads by CID it refuses. It is the gate's
// types.PolicyGateObserver.
type PolicyGate struct {
	enforcement func() types.PolicyEnforcement
	records     RecordCounter

	searchRecords *prometheus.Desc
	unevaluated   *prometheus.Desc
	fetches       *prometheus.CounterVec
}

// NewPolicyGate reports on the gate applying what enforcement returns.
func NewPolicyGate(enforcement func() types.PolicyEnforcement, records RecordCounter) *PolicyGate {
	fetches := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "dir_policy_gate_fetches_excluded_total",
		Help: `Reads of a record by CID the enforced content policies excluded (outcome="excluded") or, in shadow mode, would have (outcome="would_exclude").`,
	}, []string{"outcome"})

	// Start both series at zero, so a rate over them is defined before the
	// first exclusion.
	fetches.WithLabelValues(outcomeExcluded)
	fetches.WithLabelValues(outcomeWouldExclude)

	return &PolicyGate{
		enforcement: enforcement,
		records:     records,
		searchRecords: prometheus.NewDesc(
			"dir_policy_gate_search_records_excluded",
			`Indexed records searches exclude for failing a content policy: those the policies being enforced leave out (outcome="excluded"), and those every policy being checked would leave out if all of them were enforced (outcome="would_exclude"). The second is reported only when something is checked in shadow mode, and is never below the first.`,
			[]string{"outcome"}, nil,
		),
		unevaluated: prometheus.NewDesc(
			"dir_policy_gate_records_unevaluated",
			"Indexed records with no verdict yet under the current version of a content policy. Enforcing reads hide them until the reconciler has evaluated them, so this is the size of the blackout after a policy is edited, and what remains before a newly added policy is first enforced.",
			[]string{"policy_id", "version"}, nil,
		),
		fetches: fetches,
	}
}

// FetchExcluded counts a read by CID the gate excluded, or would have.
func (g *PolicyGate) FetchExcluded(mode policyconfig.Mode) {
	if outcome, ok := outcomeOf(mode); ok {
		g.fetches.WithLabelValues(outcome).Inc()
	}
}

// Describe implements prometheus.Collector.
func (g *PolicyGate) Describe(ch chan<- *prometheus.Desc) {
	g.fetches.Describe(ch)

	ch <- g.searchRecords

	ch <- g.unevaluated
}

// Collect implements prometheus.Collector. The search gauge is left out while
// searches check no policy; see collectSearchRecords for when each of its two
// series is reported.
func (g *PolicyGate) Collect(ch chan<- prometheus.Metric) {
	g.fetches.Collect(ch)

	enforcement := g.enforcement()

	for _, policy := range append(append([]types.EnforcedPolicy(nil), enforcement.Policies...), enforcement.Pending...) {
		if policy.Version == "" {
			continue
		}

		remaining, err := g.records.CountRecordsNeedingPolicyEvaluation(policy.ID, policy.Version)
		if err != nil {
			logger.Warn("Could not count records without a verdict", "policy_id", policy.ID, "error", err)

			continue
		}

		ch <- prometheus.MustNewConstMetric(g.unevaluated, prometheus.GaugeValue, float64(remaining), policy.ID, policy.Version)
	}

	g.collectSearchRecords(ch, enforcement)
}

// collectSearchRecords reports what searches leave out.
//
// Two sets, because the enforcement block and each search policy carry their
// own mode: what is applied, counted as "excluded", and everything checked,
// counted as "would_exclude". The second is reported only when something is
// checked without being applied, so that with a single mode in play — the
// only arrangement before search policies existed — exactly one series is
// emitted, as before.
func (g *PolicyGate) collectSearchRecords(ch chan<- prometheus.Metric, enforcement types.PolicyEnforcement) {
	var applied []types.EnforcedPolicy
	if enforcement.Search == policyconfig.ModeEnforce {
		applied = enforcement.Policies
	}

	appliedSearch := enforcement.EnforcingSearchPolicies()

	if len(applied) > 0 || len(appliedSearch) > 0 {
		g.countExcluded(ch, outcomeExcluded, applied, appliedSearch)
	}

	var checked []types.EnforcedPolicy
	if enforcement.Search.Active() {
		checked = enforcement.Policies
	}

	if len(checked) > len(applied) || len(enforcement.SearchPolicies) > len(appliedSearch) {
		g.countExcluded(ch, outcomeWouldExclude, checked, enforcement.SearchPolicies)
	}
}

// countExcluded reports the records a search gated on these policies leaves
// out. A count that fails is left out rather than failing the whole scrape.
func (g *PolicyGate) countExcluded(ch chan<- prometheus.Metric, outcome string,
	policies []types.EnforcedPolicy, search []types.SearchPolicy,
) {
	excluded, err := g.records.CountRecordsExcluded(policies, search)
	if err != nil {
		logger.Warn("Could not count records excluded by enforced policies", "outcome", outcome, "error", err)

		return
	}

	ch <- prometheus.MustNewConstMetric(g.searchRecords, prometheus.GaugeValue, float64(excluded), outcome)
}

// outcomes labels what a read in each checking mode excludes.
var outcomes = map[policyconfig.Mode]string{
	policyconfig.ModeShadow:  outcomeWouldExclude,
	policyconfig.ModeEnforce: outcomeExcluded,
}

// outcomeOf labels what a read in mode excludes; a mode that checks nothing
// has no label.
func outcomeOf(mode policyconfig.Mode) (string, bool) {
	outcome, ok := outcomes[mode]

	return outcome, ok
}
