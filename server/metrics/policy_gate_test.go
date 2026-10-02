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

// excludedRecords answers the counts: count records are excluded, and
// unevaluated lack a verdict.
type excludedRecords struct {
	count       int64
	unevaluated int64
	err         error
}

func (r excludedRecords) CountRecordsExcluded([]types.EnforcedPolicy) (int64, error) {
	return r.count, r.err
}

func (r excludedRecords) CountRecordsNeedingPolicyEvaluation(string, string) (int64, error) {
	return r.unevaluated, r.err
}

func gateFor(search policyconfig.Mode, records excludedRecords) *PolicyGate {
	return NewPolicyGate(func() types.PolicyEnforcement {
		return types.PolicyEnforcement{
			Policies: []types.EnforcedPolicy{{ID: "opa:a", Version: "v1"}},
			Search:   search,
		}
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

// The search gauge tells "would exclude" from "excluded" by the search mode,
// and is absent while searches check nothing.
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

// A failed count leaves the gauge out without failing the scrape.
func TestPolicyGate_CountErrorDropsOnlyTheGauge(t *testing.T) {
	t.Parallel()

	gate := gateFor(policyconfig.ModeEnforce, excludedRecords{err: errors.New("database unavailable")})

	assert.Empty(t, gather(t, gate, "dir_policy_gate_search_records_excluded"))
	assert.Len(t, gather(t, gate, "dir_policy_gate_fetches_excluded_total"), 2)
}

// Fetches are counted under the outcome of the mode that saw them; both
// series exist from the start.
func TestPolicyGate_CountsExcludedFetchesByOutcome(t *testing.T) {
	t.Parallel()

	gate := gateFor(policyconfig.ModeOff, excludedRecords{})
	assert.Equal(t, map[string]float64{outcomeExcluded: 0, outcomeWouldExclude: 0},
		gather(t, gate, "dir_policy_gate_fetches_excluded_total"))

	gate.FetchExcluded(policyconfig.ModeEnforce)
	gate.FetchExcluded(policyconfig.ModeShadow)
	gate.FetchExcluded(policyconfig.ModeShadow)
	gate.FetchExcluded(policyconfig.ModeOff)

	assert.Equal(t, map[string]float64{outcomeExcluded: 1, outcomeWouldExclude: 2},
		gather(t, gate, "dir_policy_gate_fetches_excluded_total"))
}

// Each registered policy reports how many records have no verdict yet under
// its current version: the size of the blackout after an edit. A policy no
// evaluator registered has no version to count under.
func TestPolicyGate_ReportsUnevaluatedRecordsPerPolicy(t *testing.T) {
	t.Parallel()

	gate := NewPolicyGate(func() types.PolicyEnforcement {
		return types.PolicyEnforcement{
			Policies: []types.EnforcedPolicy{{ID: "opa:done", Version: "v1"}},
			Pending:  []types.EnforcedPolicy{{ID: "opa:new", Version: "v1"}, {ID: "opa:unregistered"}},
		}
	}, excludedRecords{unevaluated: 12})

	assert.Equal(t, map[string]float64{"opa:done/v1": 12, "opa:new/v1": 12}, gather(t, gate, "dir_policy_gate_records_unevaluated"))
}

// A policy whose records cannot be counted is left out without failing the
// scrape.
func TestPolicyGate_UnevaluatedCountErrorDropsOnlyThatGauge(t *testing.T) {
	t.Parallel()

	gate := NewPolicyGate(func() types.PolicyEnforcement {
		return types.PolicyEnforcement{Policies: []types.EnforcedPolicy{{ID: "opa:a", Version: "v1"}}}
	}, excludedRecords{err: errors.New("database unavailable")})

	assert.Empty(t, gather(t, gate, "dir_policy_gate_records_unevaluated"))
	assert.Len(t, gather(t, gate, "dir_policy_gate_fetches_excluded_total"), 2)
}
