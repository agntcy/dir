// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package publication

import (
	"context"
	"errors"
	"testing"

	corev1 "github.com/agntcy/dir/api/core/v1"
	routingv1 "github.com/agntcy/dir/api/routing/v1"
	"github.com/agntcy/dir/server/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fakePublicationWorkerDatabase struct {
	types.DatabaseAPI

	cids []string
	err  error
}

func (f *fakePublicationWorkerDatabase) GetRecordCIDs(
	_ ...types.FilterOption,
) ([]string, error) {
	return append([]string(nil), f.cids...), f.err
}

func allRecordsRequest(value bool) *routingv1.PublishRequest {
	return &routingv1.PublishRequest{
		Request: &routingv1.PublishRequest_AllRecords{
			AllRecords: value,
		},
	}
}

func TestGetCIDsFromAllRecordsRequest(t *testing.T) {
	database := &fakePublicationWorkerDatabase{
		cids: []string{"stored-a", "already-published", "stored-b"},
	}
	worker := &Worker{
		db: database,
	}

	cids, err := worker.getCIDsFromRequest(
		context.Background(), allRecordsRequest(true))

	require.NoError(t, err)
	assert.Equal(
		t,
		[]string{"stored-a", "already-published", "stored-b"},
		cids,
	)
}

func TestGetCIDsFromAllRecordsRequestErrors(t *testing.T) {
	t.Run("rejects false all_records", func(t *testing.T) {
		worker := &Worker{
			db: &fakePublicationWorkerDatabase{},
		}

		_, err := worker.getCIDsFromRequest(
			context.Background(), allRecordsRequest(false))

		require.Error(t, err)
		require.ErrorContains(t, err, "all_records must be true")
	})

	t.Run("database failure", func(t *testing.T) {
		worker := &Worker{
			db: &fakePublicationWorkerDatabase{
				err: errors.New("database unavailable"),
			},
		}

		_, err := worker.getCIDsFromRequest(
			context.Background(), allRecordsRequest(true))

		require.Error(t, err)
		require.ErrorContains(t, err, "failed to list stored records")
	})
}

const announceCID = "baeareidp4vt6jw7tirdvk6qlcuqndobz24yejxovcjgcuv3qnnhwzqz4mi"

// announceGateDB answers the policy gate.
type announceGateDB struct {
	types.DatabaseAPI

	servable bool
	err      error
}

func (d *announceGateDB) IsRecordServable(string) (bool, error) { return d.servable, d.err }

// announceStore holds at most the record announceCID and counts pulls.
type announceStore struct {
	types.StoreAPI

	present bool
	pulls   int
}

func (s *announceStore) Pull(_ context.Context, ref *corev1.RecordRef) (*corev1.Record, error) {
	s.pulls++

	if !s.present {
		return nil, types.RecordNotFoundError(ref.GetCid()) //nolint:wrapcheck // the store's own not-found status
	}

	return &corev1.Record{}, nil
}

// An excluded record is not announced, and publishing it fails with the
// status that says so. A record that passes the gate but is not in the store
// fails with not-found instead.
func TestAnnounceToDHT_ExcludedRecordIsReportedAsExcluded(t *testing.T) {
	t.Parallel()

	exStore := &announceStore{present: true}
	exErr := (&Worker{db: &announceGateDB{servable: false}, store: exStore}).announceToDHT(t.Context(), announceCID)

	missErr := (&Worker{db: &announceGateDB{servable: true}, store: &announceStore{present: false}}).announceToDHT(t.Context(), announceCID)

	require.Error(t, exErr)
	require.Error(t, missErr)
	assert.Equal(t, codes.PermissionDenied, status.Code(exErr))
	assert.Equal(t, codes.NotFound, status.Code(missErr))
	assert.Zero(t, exStore.pulls, "an excluded record must not be read to be announced")
}

// A gate error blocks the announcement even if the answer it came with says
// servable, and is reported as a failed check, not as an excluded record.
func TestAnnounceToDHT_GateErrorAnnouncesNothing(t *testing.T) {
	t.Parallel()

	store := &announceStore{present: true}
	db := &announceGateDB{servable: true, err: errors.New("database unavailable")}
	err := (&Worker{db: db, store: store}).announceToDHT(t.Context(), announceCID)

	require.ErrorContains(t, err, "check record against enforced policies")
	assert.Zero(t, store.pulls)
}
