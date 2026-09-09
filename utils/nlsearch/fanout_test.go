// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package nlsearch

import (
	"context"
	"errors"
	"sync"
	"testing"

	searchv1 "github.com/agntcy/dir/api/search/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubSearcher answers queries from a canned map keyed by "<queryType>:<value>",
// recording what it was asked. Concurrent by contract, so it locks.
type stubSearcher struct {
	byQuery map[string][]string
	err     map[string]error

	mu        sync.Mutex
	gotLimits []uint32
	gotKeys   []string
}

func key(q *searchv1.RecordQuery) string {
	return q.GetType().String() + ":" + q.GetValue()
}

func (s *stubSearcher) SearchCIDs(_ context.Context, q *searchv1.RecordQuery, limit uint32) ([]string, error) {
	s.mu.Lock()
	s.gotLimits = append(s.gotLimits, limit)
	s.gotKeys = append(s.gotKeys, key(q))
	s.mu.Unlock()

	if err, ok := s.err[key(q)]; ok {
		return nil, err
	}

	return s.byQuery[key(q)], nil
}

func skillSignal(name string, score float64) Signal {
	return Signal{Type: SignalTypeSkillName, Value: name, Score: score}
}

func domainSignal(name string, score float64) Signal {
	return Signal{Type: SignalTypeDomainName, Value: name, Score: score}
}

func TestFanOutAndScoreRanksByHitCount(t *testing.T) {
	s := &stubSearcher{byQuery: map[string][]string{
		"RECORD_QUERY_TYPE_SKILL_NAME:code_review":     {"a", "b"},
		"RECORD_QUERY_TYPE_DOMAIN_NAME:software_dev":   {"b", "c"},
		"RECORD_QUERY_TYPE_SKILL_NAME:static_analysis": {"b"},
	}}

	got := FanOutAndScore(context.Background(), []Signal{
		skillSignal("code_review", 0.9),
		domainSignal("software_dev", 0.8),
		skillSignal("static_analysis", 0.7),
	}, s, FanOutOptions{})

	// Union of all signals, not an intersection: "a" and "c" matched one signal
	// each and still appear. "b" matched all three, so it ranks first.
	assert.Equal(t, []string{"b", "a", "c"}, got.CIDs)
	assert.Equal(t, 3, got.HitCount["b"])
	assert.Equal(t, 1, got.HitCount["a"])
	assert.Len(t, got.CIDSignals["b"], 3)
}

func TestFanOutAndScoreTieBreaksDeterministically(t *testing.T) {
	// Every CID matches exactly one signal, so hit count cannot separate them:
	// score then CID must, and identically on every run.
	s := &stubSearcher{byQuery: map[string][]string{
		"RECORD_QUERY_TYPE_SKILL_NAME:low":  {"zzz", "aaa"},
		"RECORD_QUERY_TYPE_SKILL_NAME:high": {"mmm", "bbb"},
	}}
	signals := []Signal{skillSignal("low", 0.2), skillSignal("high", 0.9)}

	first := FanOutAndScore(context.Background(), signals, s, FanOutOptions{}).CIDs

	// Higher-scoring signal's hits sort first; within a signal, CID ascending.
	assert.Equal(t, []string{"bbb", "mmm", "aaa", "zzz"}, first)

	for range 25 {
		assert.Equal(t, first, FanOutAndScore(context.Background(), signals, s, FanOutOptions{}).CIDs,
			"ranking must not depend on which query finishes first")
	}
}

func TestFanOutAndScoreKeywordUnionsNameAndDescription(t *testing.T) {
	s := &stubSearcher{byQuery: map[string][]string{
		"RECORD_QUERY_TYPE_NAME:triage":        {"a", "b"},
		"RECORD_QUERY_TYPE_DESCRIPTION:triage": {"b", "c"},
	}}

	got := FanOutAndScore(context.Background(),
		[]Signal{{Type: SignalTypeKeyword, Value: "triage", Score: 0.5}}, s, FanOutOptions{})

	assert.ElementsMatch(t, []string{"a", "b", "c"}, got.CIDs)
	// "b" matched both fields but the keyword is one signal, so it counts once —
	// otherwise a keyword would outweigh a taxonomy signal.
	assert.Equal(t, 1, got.HitCount["b"])
	assert.Equal(t, 3, got.PerSignal[0].Count, "the keyword reports its deduped union size")
}

func TestFanOutAndScoreSignalErrorDegradesRecall(t *testing.T) {
	boom := errors.New("backend down")
	s := &stubSearcher{
		byQuery: map[string][]string{"RECORD_QUERY_TYPE_SKILL_NAME:ok": {"a"}},
		err:     map[string]error{"RECORD_QUERY_TYPE_DOMAIN_NAME:bad": boom},
	}

	got := FanOutAndScore(context.Background(), []Signal{
		skillSignal("ok", 0.9),
		domainSignal("bad", 0.8),
	}, s, FanOutOptions{})

	// The healthy signal still contributes; the failure is reported, not fatal.
	assert.Equal(t, []string{"a"}, got.CIDs)
	require.Len(t, got.PerSignal, 2)
	require.NoError(t, got.PerSignal[0].Err)
	require.Error(t, got.PerSignal[1].Err)
	assert.Equal(t, "bad", got.PerSignal[1].Signal.Value, "PerSignal follows the input order")
}

func TestFanOutAndScorePerSignalLimit(t *testing.T) {
	s := &stubSearcher{byQuery: map[string][]string{}}

	FanOutAndScore(context.Background(), []Signal{skillSignal("x", 0.5)}, s, FanOutOptions{})
	require.Equal(t, []uint32{DefaultFanOutLimit}, s.gotLimits)

	s.gotLimits = nil
	FanOutAndScore(context.Background(), []Signal{skillSignal("x", 0.5)}, s, FanOutOptions{PerSignalLimit: 7})
	assert.Equal(t, []uint32{7}, s.gotLimits)
}

func TestFanOutAndScoreNoSignals(t *testing.T) {
	s := &stubSearcher{byQuery: map[string][]string{}}

	got := FanOutAndScore(context.Background(), nil, s, FanOutOptions{})

	assert.Empty(t, got.CIDs)
	assert.Empty(t, got.PerSignal)
	assert.Empty(t, s.gotKeys, "no signals means no queries")
}
