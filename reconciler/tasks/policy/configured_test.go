// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package policy

import (
	"context"
	"testing"

	typesv1alpha1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/agntcy/oasf/types/v1alpha1"
	coretypes "github.com/agntcy/dir/api/core/types"
	corev1 "github.com/agntcy/dir/api/core/v1"
	serverpolicy "github.com/agntcy/dir/server/policy"
	policyconfig "github.com/agntcy/dir/server/policy/config"
	servertypes "github.com/agntcy/dir/server/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// recordSource serves records by CID and counts the reads.
type recordSource struct {
	records map[string]*corev1.Record
	err     error
	pulls   []string
}

func (s *recordSource) Pull(_ context.Context, ref *corev1.RecordRef) (*corev1.Record, error) {
	s.pulls = append(s.pulls, ref.GetCid())

	if s.err != nil {
		return nil, s.err
	}

	record, ok := s.records[ref.GetCid()]
	if !ok {
		return nil, servertypes.RecordNotFoundError(ref.GetCid()) //nolint:wrapcheck // the store's own status
	}

	return record, nil
}

func oasfRecord(name, description string) *corev1.Record {
	return corev1.New(&typesv1alpha1.Record{Name: name, Description: description})
}

// A real CEL policy, run by the task: the verdicts reach the database with the
// version the policy was loaded at, and a record the policy rejects says why.
func TestTask_RunsAConfiguredPolicy(t *testing.T) {
	t.Parallel()

	registry, err := serverpolicy.NewRegistry(t.Context(), policyconfig.Evaluations{{
		Provider:    policyconfig.ProviderCEL,
		Name:        "has-description",
		Expressions: []string{`record.description != ""`},
	}}, nil, "")
	require.NoError(t, err)

	src := &recordSource{records: map[string]*corev1.Record{
		"cid-described": oasfRecord("described", "does something"),
		"cid-bare":      oasfRecord("bare", ""),
	}}
	evaluators := registry.Evaluators(src, nil)
	require.Len(t, evaluators, 1)

	db := &fakeDB{records: map[string][]coretypes.Record{evaluators[0].PolicyID(): records("cid-bare", "cid-described")}}

	task, err := NewTask(Config{Enabled: true}, db, evaluators...)
	require.NoError(t, err)
	require.True(t, task.IsEnabled())

	require.NoError(t, task.Run(t.Context()))

	assert.Equal(t, []string{evaluators[0].PolicyID() + "@" + evaluators[0].PolicyVersion()}, db.registered)

	described := storedFor(t, db, "cid-described")
	assert.True(t, described.Compliant)
	assert.Equal(t, evaluators[0].PolicyVersion(), described.PolicyVersion)

	bare := storedFor(t, db, "cid-bare")
	assert.False(t, bare.Compliant)
	assert.Equal(t, servertypes.PolicyEvalStatusEvaluated, bare.Status, "rejected is a verdict, not a failure")
	assert.Contains(t, bare.Reason, "record.description")
}
