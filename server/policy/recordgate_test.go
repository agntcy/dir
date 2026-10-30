// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package policy

import (
	"errors"
	"testing"

	"github.com/agntcy/dir/server/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// servability is a stand-in for the index the enforced verdicts are read from.
type servability struct {
	types.RecordServabilityAPI

	servable bool
	err      error
}

func (s servability) IsRecordServable(string) (bool, error) { return s.servable, s.err }

const gatedCID = "baeareigatetest00000000000000000000000000000000000000000000000"

// Both halves of the gate can refuse, and a caller is told the same thing
// either way: the record is not available under this node's content policy,
// with no policy name and no reason.
func TestRecordGateCheck_EitherHalfCanRefuse(t *testing.T) {
	t.Parallel()

	refusing := registryReading(t, `verdicts["cel:trusted"].compliant_and_current`,
		&fakeVerdicts{}, enforcing(trustedPolicy))
	allowing := registryReading(t, `record_cid != ""`, &fakeVerdicts{}, enforcing(trustedPolicy))

	for _, tc := range []struct {
		name string
		gate *RecordGate
		code codes.Code
	}{
		{"both allow", NewRecordGate(servability{servable: true}, allowing), codes.OK},
		{"the index refuses", NewRecordGate(servability{servable: false}, allowing), codes.PermissionDenied},
		{"the policy refuses", NewRecordGate(servability{servable: true}, refusing), codes.PermissionDenied},
		{"neither is configured", NewRecordGate(nil, nil), codes.OK},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := tc.gate.Check(t.Context(), gatedCID)

			assert.Equal(t, tc.code, status.Code(err))
		})
	}
}

// A check that cannot be answered fails closed, and is reported as a failed
// check rather than as an excluded record: the two mean different things to
// whoever is reading the logs.
func TestRecordGateCheck_AnUnanswerableCheckFailsClosed(t *testing.T) {
	t.Parallel()

	gate := NewRecordGate(servability{servable: true, err: errors.New("database unavailable")}, nil)

	err := gate.Check(t.Context(), gatedCID)

	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

// The index is asked first, so a record already excluded there costs no
// policy evaluation.
func TestRecordGateCheck_AnExcludedRecordIsNotEvaluated(t *testing.T) {
	t.Parallel()

	source := &fakeVerdicts{}
	gate := NewRecordGate(servability{servable: false},
		registryReading(t, `verdicts["cel:trusted"].compliant_and_current`, source, enforcing(trustedPolicy)))

	require.Error(t, gate.Check(t.Context(), gatedCID))

	assert.Empty(t, source.cids, "the verdicts of an excluded record must not be read")
}
