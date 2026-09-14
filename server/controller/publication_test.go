// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"testing"

	routingv1 "github.com/agntcy/dir/api/routing/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fakePublicationService struct {
	request *routingv1.PublishRequest
	id      string
	err     error
}

func (f *fakePublicationService) CreatePublication(
	_ context.Context,
	request *routingv1.PublishRequest,
) (string, error) {
	f.request = request

	return f.id, f.err
}

func TestCreatePublicationAcceptsAllRecords(t *testing.T) {
	publication := &fakePublicationService{id: "publication-id"}
	controller := NewPublicationController(nil, publication, nil)

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
	require.NotNil(t, publication.request)
	assert.True(t, publication.request.GetAllRecords())
}

func TestCreatePublicationRejectsFalseAllRecords(t *testing.T) {
	publication := &fakePublicationService{}
	controller := NewPublicationController(nil, publication, nil)

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
	assert.Nil(t, publication.request)
}
