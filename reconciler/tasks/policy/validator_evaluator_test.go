// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package policy

import (
	"context"
	"errors"
	"testing"

	typesv1alpha1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/agntcy/oasf/types/v1alpha1"
	coretypes "github.com/agntcy/dir/api/core/types"
	corev1 "github.com/agntcy/dir/api/core/v1"
	servertypes "github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/server/validators"
	validatorsconfig "github.com/agntcy/dir/server/validators/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
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

// funcValidator is a corev1.Validator whose verdicts come from validate.
type funcValidator func(*structpb.Struct) (bool, []string, []string, error)

func (f funcValidator) ValidateRecord(_ context.Context, data *structpb.Struct) (bool, []string, []string, error) {
	return f(data)
}

func oasfRecord(name, description string) *corev1.Record {
	return corev1.New(&typesv1alpha1.Record{Name: name, Description: description})
}

func TestValidatorEvaluator_Evaluate(t *testing.T) {
	t.Parallel()

	accept := funcValidator(func(*structpb.Struct) (bool, []string, []string, error) { return true, nil, nil, nil })
	reject := funcValidator(func(*structpb.Struct) (bool, []string, []string, error) {
		return false, []string{"no licence", "no owner"}, nil, nil
	})
	broken := funcValidator(func(*structpb.Struct) (bool, []string, []string, error) {
		return false, nil, nil, errors.New("engine failed")
	})

	t.Run("a record the validator accepts complies", func(t *testing.T) {
		t.Parallel()

		src := &recordSource{records: map[string]*corev1.Record{"cid-a": oasfRecord("a", "")}}
		ev := NewValidatorEvaluator("opa:p", "v1", accept, src)

		compliant, reason, err := ev.Evaluate(t.Context(), &fakeRecord{cid: "cid-a"})

		require.NoError(t, err)
		assert.True(t, compliant)
		assert.Empty(t, reason)
		assert.Equal(t, []string{"cid-a"}, src.pulls, "the record is read from the store by its CID")
	})

	t.Run("a record the validator rejects does not, and the reason is what it said", func(t *testing.T) {
		t.Parallel()

		src := &recordSource{records: map[string]*corev1.Record{"cid-a": oasfRecord("a", "")}}

		compliant, reason, err := NewValidatorEvaluator("opa:p", "v1", reject, src).Evaluate(t.Context(), &fakeRecord{cid: "cid-a"})

		require.NoError(t, err)
		assert.False(t, compliant)
		assert.Equal(t, "ERROR: no licence; ERROR: no owner", reason)
	})

	t.Run("a record that cannot be read is an error, not a verdict", func(t *testing.T) {
		t.Parallel()

		ev := NewValidatorEvaluator("opa:p", "v1", accept, &recordSource{})

		compliant, _, err := ev.Evaluate(t.Context(), &fakeRecord{cid: "missing"})

		require.ErrorContains(t, err, "read record missing")
		assert.False(t, compliant)
	})

	t.Run("a validator that fails is an error, not a verdict", func(t *testing.T) {
		t.Parallel()

		src := &recordSource{records: map[string]*corev1.Record{"cid-a": oasfRecord("a", "")}}

		compliant, _, err := NewValidatorEvaluator("opa:p", "v1", broken, src).Evaluate(t.Context(), &fakeRecord{cid: "cid-a"})

		require.ErrorContains(t, err, "engine failed")
		assert.False(t, compliant)
	})

	t.Run("it reports the policy it was given", func(t *testing.T) {
		t.Parallel()

		ev := NewValidatorEvaluator("opa:p", "v1", accept, &recordSource{})

		assert.Equal(t, "opa:p", ev.PolicyID())
		assert.Equal(t, "v1", ev.PolicyVersion())
	})
}

// A real CEL policy, run by the task: the verdicts reach the database with the
// version the policy was loaded at, and a record the policy rejects says why.
func TestTask_RunsAConfiguredPolicy(t *testing.T) {
	t.Parallel()

	registry, err := validators.NewRegistry(t.Context(), validatorsconfig.Config{{
		Provider: validatorsconfig.ProviderCEL,
		Ops:      []string{validatorsconfig.OpEvaluate},
		Config: map[string]any{
			validatorsconfig.ConfigKeyName:        "has-description",
			validatorsconfig.ConfigKeyExpressions: []string{`record.description != ""`},
		},
	}}, "")
	require.NoError(t, err)

	policies := registry.Policies()
	require.Len(t, policies, 1)

	src := &recordSource{records: map[string]*corev1.Record{
		"cid-described": oasfRecord("described", "does something"),
		"cid-bare":      oasfRecord("bare", ""),
	}}
	db := &fakeDB{records: map[string][]coretypes.Record{policies[0].ID: records("cid-bare", "cid-described")}}

	task, err := NewTask(Config{Enabled: true}, db, NewValidatorEvaluator(policies[0].ID, policies[0].Version, policies[0].Validator, src))
	require.NoError(t, err)
	require.True(t, task.IsEnabled())

	require.NoError(t, task.Run(t.Context()))

	assert.Equal(t, []string{"cel:has-description@" + policies[0].Version}, db.registered)

	described := storedFor(t, db, "cid-described")
	assert.True(t, described.Compliant)
	assert.Equal(t, policies[0].Version, described.PolicyVersion)

	bare := storedFor(t, db, "cid-bare")
	assert.False(t, bare.Compliant)
	assert.Equal(t, servertypes.PolicyEvalStatusEvaluated, bare.Status, "rejected is a verdict, not a failure")
	assert.Contains(t, bare.Reason, "record.description")
}
