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
	CountRecordsExcluded(policies []types.EnforcedPolicy) (int64, error)
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
			`Indexed records searches exclude (outcome="excluded") or, in shadow mode, would (outcome="would_exclude"), for not complying with every enforced content policy.`,
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
// searches check no policy, and when it cannot be counted: failing it would
// fail the whole scrape.
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

	outcome, ok := outcomeOf(enforcement.Search)
	if !ok || len(enforcement.Policies) == 0 {
		return
	}

	excluded, err := g.records.CountRecordsExcluded(enforcement.Policies)
	if err != nil {
		logger.Warn("Could not count records excluded by enforced policies", "error", err)

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
