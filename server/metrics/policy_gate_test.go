// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package metrics

import (
	"errors"
	"strings"
	"testing"

	policyconfig "github.com/agntcy/dir/server/policy/config"
	"github.com/agntcy/dir/server/types"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// excludedRecords answers the counts: count records are excluded per policy
// asked about, and unevaluated lack a verdict.
//
// Per policy so that the number a series carries says which set was counted,
// which is the only thing separating the two search outcomes.
type excludedRecords struct {
	count       int64
	unevaluated int64
	err         error
}

func (r excludedRecords) CountRecordsExcluded(policies []types.SearchPolicy) (int64, error) {
	return r.count * int64(len(policies)), r.err
}

func (r excludedRecords) CountRecordsNeedingPolicyEvaluation(string, string) (int64, error) {
	return r.unevaluated, r.err
}

// gateFor reports one search policy in mode, and one evaluation policy for
// the unevaluated gauge to ask about.
func gateFor(mode policyconfig.Mode, records excludedRecords) *PolicyGate {
	return NewPolicyGate(func() types.PolicyEnforcement {
		enforcement := types.PolicyEnforcement{
			Evaluations: map[string]types.ResolvedPolicy{
				"opa:a": {VersionedPolicy: types.VersionedPolicy{ID: "opa:a", Version: "v1"}, Covered: true},
			},
		}
		if mode != policyconfig.ModeOff {
			enforcement.SearchPolicies = []types.SearchPolicy{{ID: "cel:served", Mode: mode}}
		}

		return enforcement
	}, records)
}

// gather scrapes gate through a registry that checks its descriptions, and
// returns each series of metric by its label values, joined by "/".
func gather(t *testing.T, gate *PolicyGate, metric string) map[string]float64 {
	t.Helper()

	registry := prometheus.NewPedanticRegistry()
	require.NoError(t, registry.Register(gate))

	families, err := registry.Gather()
	require.NoError(t, err)

	series := map[string]float64{}

	for _, family := range families {
		if family.GetName() != metric {
			continue
		}

		for _, m := range family.GetMetric() {
			labels := make([]string, 0, len(m.GetLabel()))
			for _, label := range m.GetLabel() {
				labels = append(labels, label.GetValue())
			}

			series[strings.Join(labels, "/")] = m.GetGauge().GetValue() + m.GetCounter().GetValue()
		}
	}

	return series
}

// The search gauge tells "would exclude" from "excluded" by the policy's
// mode, and is absent while searches check nothing.
func TestPolicyGate_SearchGaugeFollowsTheMode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		mode policyconfig.Mode
		want map[string]float64
	}{
		{policyconfig.ModeShadow, map[string]float64{outcomeWouldExclude: 7}},
		{policyconfig.ModeEnforce, map[string]float64{outcomeExcluded: 7}},
		{policyconfig.ModeOff, map[string]float64{}},
	}

	for _, tt := range tests {
		t.Run(string(tt.mode), func(t *testing.T) {
			t.Parallel()

			got := gather(t, gateFor(tt.mode, excludedRecords{count: 7}), "dir_policy_gate_search_records_excluded")
			assert.Equal(t, tt.want, got)
		})
	}
}

// A search policy is counted under the outcome of its own mode.
func TestPolicyGate_SearchPolicyIsCountedUnderItsOwnMode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		mode policyconfig.Mode
		want map[string]float64
	}{
		{"enforcing", policyconfig.ModeEnforce, map[string]float64{outcomeExcluded: 1}},
		{"shadow", policyconfig.ModeShadow, map[string]float64{outcomeWouldExclude: 1}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gate := NewPolicyGate(func() types.PolicyEnforcement {
				return types.PolicyEnforcement{
					SearchPolicies: []types.SearchPolicy{{ID: "cel:served", Mode: tt.mode}},
				}
			}, excludedRecords{count: 1})

			assert.Equal(t, tt.want, gather(t, gate, "dir_policy_gate_search_records_excluded"))
		})
	}
}

// With something enforcing and something else in shadow, both series are
// reported: what is left out now, and what would be if the shadowed policy
// were turned on too. The gap between them is the cost of that switch.
func TestPolicyGate_ShadowIsCountedOnTopOfWhatIsEnforced(t *testing.T) {
	t.Parallel()

	gate := NewPolicyGate(func() types.PolicyEnforcement {
		return types.PolicyEnforcement{
			SearchPolicies: []types.SearchPolicy{
				{ID: "cel:on", Mode: policyconfig.ModeEnforce},
				{ID: "cel:trying", Mode: policyconfig.ModeShadow},
			},
		}
	}, excludedRecords{count: 1})

	// One enforcing now; both would be.
	assert.Equal(t, map[string]float64{outcomeExcluded: 1, outcomeWouldExclude: 2},
		gather(t, gate, "dir_policy_gate_search_records_excluded"))
}

// A failed count leaves the gauge out without failing the scrape.
func TestPolicyGate_CountErrorDropsOnlyTheGauge(t *testing.T) {
	t.Parallel()

	gate := gateFor(policyconfig.ModeEnforce, excludedRecords{err: errors.New("database unavailable")})

	assert.Empty(t, gather(t, gate, "dir_policy_gate_search_records_excluded"))
	assert.Len(t, gather(t, gate, "dir_policy_gate_pulls_excluded_total"), 2)
}

// Pulls are counted under the outcome of the mode that saw them; both
// series exist from the start.
func TestPolicyGate_CountsExcludedPullsByOutcome(t *testing.T) {
	t.Parallel()

	gate := gateFor(policyconfig.ModeOff, excludedRecords{})
	assert.Equal(t, map[string]float64{outcomeExcluded: 0, outcomeWouldExclude: 0},
		gather(t, gate, "dir_policy_gate_pulls_excluded_total"))

	gate.PullExcluded(policyconfig.ModeEnforce)
	gate.PullExcluded(policyconfig.ModeShadow)
	gate.PullExcluded(policyconfig.ModeShadow)
	gate.PullExcluded(policyconfig.ModeOff)

	assert.Equal(t, map[string]float64{outcomeExcluded: 1, outcomeWouldExclude: 2},
		gather(t, gate, "dir_policy_gate_pulls_excluded_total"))
}

// Each registered evaluation policy reports how many records have no verdict
// yet under its current version: the size of the blackout after an edit. A
// policy no evaluator registered has no version to count under.
//
// Covered is what the two differ by, and it is not what is counted: the
// backfilling policy is precisely the one an operator is watching.
func TestPolicyGate_ReportsUnevaluatedRecordsPerPolicy(t *testing.T) {
	t.Parallel()

	gate := NewPolicyGate(func() types.PolicyEnforcement {
		return types.PolicyEnforcement{Evaluations: map[string]types.ResolvedPolicy{
			"opa:done":         resolved("opa:done", "v1", true),
			"opa:new":          resolved("opa:new", "v1", false),
			"opa:unregistered": resolved("opa:unregistered", "", false),
		}}
	}, excludedRecords{unevaluated: 12})

	assert.Equal(t, map[string]float64{"opa:done/v1": 12, "opa:new/v1": 12}, gather(t, gate, "dir_policy_gate_records_unevaluated"))
}

func resolved(id, version string, covered bool) types.ResolvedPolicy {
	return types.ResolvedPolicy{
		VersionedPolicy: types.VersionedPolicy{ID: id, Version: version}, //nolint:modernize // the type name says which half of the pair this is
		Covered:         covered,
	}
}

// A policy whose records cannot be counted is left out without failing the
// scrape.
func TestPolicyGate_UnevaluatedCountErrorDropsOnlyThatGauge(t *testing.T) {
	t.Parallel()

	gate := NewPolicyGate(func() types.PolicyEnforcement {
		return types.PolicyEnforcement{Evaluations: map[string]types.ResolvedPolicy{
			"opa:a": resolved("opa:a", "v1", true),
		}}
	}, excludedRecords{err: errors.New("database unavailable")})

	assert.Empty(t, gather(t, gate, "dir_policy_gate_records_unevaluated"))
	assert.Len(t, gather(t, gate, "dir_policy_gate_pulls_excluded_total"), 2)
}
