// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"testing"

	routingv1 "github.com/agntcy/dir/api/routing/v1"
	"github.com/agntcy/dir/server/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fakePublicationDatabase struct {
	types.DatabaseAPI

	request *routingv1.PublishRequest
	id      string
	err     error
}

func (f *fakePublicationDatabase) CreatePublication(
	request *routingv1.PublishRequest,
) (string, error) {
	f.request = request

	return f.id, f.err
}

func TestCreatePublicationAcceptsAllRecords(t *testing.T) {
	database := &fakePublicationDatabase{id: "publication-id"}
	controller := NewPublicationController(database, nil)

	response, err := controller.CreatePublication(
		context.Background(),
		&routingv1.PublishRequest{
			Request: &routingv1.PublishRequest_AllRecords{
				AllRecords: true,
			},
		},
	)

	require.NoError(t, err)
	assert.Equal(t, "publication-id", response.GetPublicationId())
	require.NotNil(t, database.request)
	assert.True(t, database.request.GetAllRecords())
}

func TestCreatePublicationRejectsFalseAllRecords(t *testing.T) {
	database := &fakePublicationDatabase{}
	controller := NewPublicationController(database, nil)

	_, err := controller.CreatePublication(
		context.Background(),
		&routingv1.PublishRequest{
			Request: &routingv1.PublishRequest_AllRecords{
				AllRecords: false,
			},
		},
	)

	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
	require.ErrorContains(t, err, "all_records must be true")
	assert.Nil(t, database.request)
}
