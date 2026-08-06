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

type fakePublicationWorkerRouting struct {
	types.RoutingAPI

	responses []*routingv1.ListResponse
	err       error
	listCalls int
}

func (f *fakePublicationWorkerRouting) List(
	_ context.Context,
	_ *routingv1.ListRequest,
) (<-chan *routingv1.ListResponse, error) {
	f.listCalls++

	if f.err != nil {
		return nil, f.err
	}

	results := make(chan *routingv1.ListResponse, len(f.responses))

	for _, response := range f.responses {
		results <- response
	}

	close(results)

	return results, nil
}

func publishedRecord(cid string) *routingv1.ListResponse {
	return &routingv1.ListResponse{
		RecordRef: &corev1.RecordRef{
			Cid: cid,
		},
	}
}

func allRecordsRequest(value bool) *routingv1.PublishRequest {
	return &routingv1.PublishRequest{
		Request: &routingv1.PublishRequest_AllRecords{
			AllRecords: value,
		},
	}
}

func TestGetCIDsFromAllRecordsRequest(t *testing.T) {
	tests := []struct {
		name          string
		stored        []string
		published     []*routingv1.ListResponse
		want          []string
		wantListCalls int
	}{
		{
			name:          "returns only unpublished records",
			stored:        []string{"stored-a", "published", "stored-b"},
			published:     []*routingv1.ListResponse{publishedRecord("published")},
			want:          []string{"stored-a", "stored-b"},
			wantListCalls: 1,
		},
		{
			name:          "returns every stored record when none are published",
			stored:        []string{"stored-a", "stored-b"},
			want:          []string{"stored-a", "stored-b"},
			wantListCalls: 1,
		},
		{
			name:   "returns empty when every stored record is published",
			stored: []string{"stored-a", "stored-b"},
			published: []*routingv1.ListResponse{
				publishedRecord("stored-a"),
				publishedRecord("stored-b"),
			},
			want:          []string{},
			wantListCalls: 1,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			database := &fakePublicationWorkerDatabase{
				cids: test.stored,
			}
			routing := &fakePublicationWorkerRouting{
				responses: test.published,
			}
			worker := &Worker{
				db:      database,
				routing: routing,
			}

			cids, err := worker.getCIDsFromRequest(
				context.Background(), allRecordsRequest(true))

			require.NoError(t, err)
			assert.Equal(t, test.want, cids)
			assert.Equal(t, test.wantListCalls, routing.listCalls)
		})
	}
}

func TestGetCIDsFromAllRecordsRequestErrors(t *testing.T) {
	t.Run("rejects false all_records", func(t *testing.T) {
		routing := &fakePublicationWorkerRouting{}
		worker := &Worker{
			db:      &fakePublicationWorkerDatabase{},
			routing: routing,
		}

		_, err := worker.getCIDsFromRequest(
			context.Background(), allRecordsRequest(false))

		require.Error(t, err)
		require.ErrorContains(t, err, "all_records must be true")
		assert.Zero(t, routing.listCalls)
	})

	t.Run("database failure", func(t *testing.T) {
		routing := &fakePublicationWorkerRouting{}
		worker := &Worker{
			db: &fakePublicationWorkerDatabase{
				err: errors.New("database unavailable"),
			},
			routing: routing,
		}

		_, err := worker.getCIDsFromRequest(
			context.Background(), allRecordsRequest(true))

		require.Error(t, err)
		require.ErrorContains(t, err, "failed to list stored records")
		assert.Zero(t, routing.listCalls)
	})

	t.Run("routing failure", func(t *testing.T) {
		routing := &fakePublicationWorkerRouting{
			err: errors.New("routing unavailable"),
		}
		worker := &Worker{
			db: &fakePublicationWorkerDatabase{
				cids: []string{"stored-a"},
			},
			routing: routing,
		}

		_, err := worker.getCIDsFromRequest(
			context.Background(), allRecordsRequest(true))

		require.Error(t, err)
		require.ErrorContains(t, err, "failed to list published records")
		assert.Equal(t, 1, routing.listCalls)
	})
}
