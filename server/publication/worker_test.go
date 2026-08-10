// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package publication

import (
	"context"
	"errors"
	"testing"

	routingv1 "github.com/agntcy/dir/api/routing/v1"
	"github.com/agntcy/dir/server/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
