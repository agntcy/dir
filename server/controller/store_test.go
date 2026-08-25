// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"io"
	"testing"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/server/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type mockStore struct {
	types.StoreAPI
	deleteErr error
}

func (m *mockStore) Delete(ctx context.Context, ref *corev1.RecordRef) error {
	return m.deleteErr
}

type mockDB struct {
	types.DatabaseAPI
	removeRecordCalled bool
}

func (m *mockDB) RemoveRecord(cid string) error {
	m.removeRecordCalled = true

	return nil
}

type mockDeleteServer struct {
	grpc.ServerStream
	reqs []*corev1.RecordRef
	idx  int
}

func (m *mockDeleteServer) Recv() (*corev1.RecordRef, error) {
	if m.idx < len(m.reqs) {
		req := m.reqs[m.idx]
		m.idx++

		return req, nil
	}

	return nil, io.EOF
}

func (m *mockDeleteServer) SendAndClose(*emptypb.Empty) error {
	return nil
}

func (m *mockDeleteServer) Context() context.Context {
	return context.Background()
}

func TestStoreDelete_NotFound(t *testing.T) {
	storeMock := &mockStore{
		deleteErr: status.Error(codes.NotFound, "record not found"),
	}
	dbMock := &mockDB{}

	ctrl := NewStoreController(storeMock, dbMock, nil, nil, nil)

	stream := &mockDeleteServer{
		reqs: []*corev1.RecordRef{{Cid: "test-cid"}},
	}

	err := ctrl.Delete(stream)
	require.NoError(t, err)
	assert.True(t, dbMock.removeRecordCalled, "RemoveRecord should be called even if store returns codes.NotFound")
}
