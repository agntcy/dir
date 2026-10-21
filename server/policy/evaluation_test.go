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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
)

type mapSource struct {
	records map[string]*corev1.Record
	err     error
	pulls   []string
}

func (s *mapSource) Pull(_ context.Context, ref *corev1.RecordRef) (*corev1.Record, error) {
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

// indexRecord is the indexed view an evaluator is handed. Only the CID is
// read; the content comes from the store.
type indexRecord struct {
	coretypes.Record

	cid string
}

func (r indexRecord) GetCid() string { return r.cid }

type funcEngine func(*structpb.Struct) (bool, []string, []string, error)

func (f funcEngine) ValidateRecord(_ context.Context, data *structpb.Struct) (bool, []string, []string, error) {
	return f(data)
}

func TestEvaluator_Evaluate(t *testing.T) {
	t.Parallel()

	accept := funcEngine(func(*structpb.Struct) (bool, []string, []string, error) { return true, nil, nil, nil })
	reject := funcEngine(func(*structpb.Struct) (bool, []string, []string, error) {
		return false, []string{"no licence", "no owner"}, nil, nil
	})
	broken := funcEngine(func(*structpb.Struct) (bool, []string, []string, error) {
		return false, nil, nil, errors.New("engine failed")
	})

	record := corev1.New(&typesv1alpha1.Record{Name: "a"})
	indexed := indexRecord{cid: record.GetCid()}

	t.Run("a record the policy accepts complies", func(t *testing.T) {
		t.Parallel()

		src := &mapSource{records: map[string]*corev1.Record{record.GetCid(): record}}
		ev := &evaluator{id: "opa:p", version: "v1", engine: accept, records: src}

		compliant, reason, err := ev.Evaluate(t.Context(), indexed)

		require.NoError(t, err)
		assert.True(t, compliant)
		assert.Empty(t, reason)
		assert.Equal(t, []string{record.GetCid()}, src.pulls, "the record is read from the store by its CID")
	})

	t.Run("a record the policy rejects does not, and the reason is what it said", func(t *testing.T) {
		t.Parallel()

		src := &mapSource{records: map[string]*corev1.Record{record.GetCid(): record}}
		ev := &evaluator{id: "opa:p", version: "v1", engine: reject, records: src}

		compliant, reason, err := ev.Evaluate(t.Context(), indexed)

		require.NoError(t, err)
		assert.False(t, compliant)
		assert.Equal(t, "ERROR: no licence; ERROR: no owner", reason)
	})

	t.Run("a record that cannot be read is an error, not a verdict", func(t *testing.T) {
		t.Parallel()

		ev := &evaluator{id: "opa:p", version: "v1", engine: accept, records: &mapSource{}}

		compliant, _, err := ev.Evaluate(t.Context(), indexed)

		require.ErrorContains(t, err, "read record "+record.GetCid())
		assert.False(t, compliant)
	})

	t.Run("an engine that fails is an error, not a verdict", func(t *testing.T) {
		t.Parallel()

		src := &mapSource{records: map[string]*corev1.Record{record.GetCid(): record}}
		ev := &evaluator{id: "opa:p", version: "v1", engine: broken, records: src}

		compliant, _, err := ev.Evaluate(t.Context(), indexed)

		require.ErrorContains(t, err, "engine failed")
		assert.False(t, compliant)
	})

	t.Run("it reports the policy it was given", func(t *testing.T) {
		t.Parallel()

		ev := &evaluator{id: "opa:p", version: "v1", engine: accept, records: &mapSource{}}

		assert.Equal(t, "opa:p", ev.PolicyID())
		assert.Equal(t, "v1", ev.PolicyVersion())
	})
}
