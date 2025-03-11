// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"fmt"
	"io"
	"log"

	storetypes "github.com/agntcy/dir/api/store/v1alpha1"
	"github.com/agntcy/dir/server/types"

	"google.golang.org/protobuf/types/known/emptypb"
)

type storeController struct {
	store types.StoreAPI
	storetypes.UnimplementedStoreServiceServer
}

func NewStoreController(store types.StoreAPI) storetypes.StoreServiceServer {
	return &storeController{
		store:                           store,
		UnimplementedStoreServiceServer: storetypes.UnimplementedStoreServiceServer{},
	}
}

func (s storeController) Push(stream storetypes.StoreService_PushServer) error {
	firstMessage, err := stream.Recv()
	if err != nil {
		return fmt.Errorf("failed to receive first message: %w", err)
	}

	log.Printf("Received: Type=%v, Name=%s\n", firstMessage.Type, firstMessage.Name)

	pr, pw := io.Pipe()

	go func() {
		defer pw.Close()
		if len(firstMessage.Data) > 0 {
			if _, err := pw.Write(firstMessage.Data); err != nil {
				pw.CloseWithError(err)
				return
			}
		}

		for {
			obj, err := stream.Recv()
			if err == io.EOF {
				return
			}

			if err != nil {
				pw.CloseWithError(err)
				return
			}

			if _, err := pw.Write(obj.Data); err != nil {
				pw.CloseWithError(err)
				return
			}
		}
	}()

	ref, err := s.store.Push(
		context.Background(),
		&storetypes.ObjectRef{
			Name: &firstMessage.Name,
			Type: &firstMessage.Type,
			Size: &firstMessage.Size,
		},
		pr,
	)
	if err != nil {
		return fmt.Errorf("failed to push: %w", err)
	}

	return stream.SendAndClose(ref)
}

func (s storeController) Pull(req *storetypes.ObjectRef, stream storetypes.StoreService_PullServer) error {
	if req.GetDigest() == "" {
		return fmt.Errorf("digest is required")
	}

	ref, err := s.store.Lookup(context.Background(), req)
	if err != nil {
		return fmt.Errorf("failed to pull: %w", err)
	}

	reader, err := s.store.Pull(context.Background(), ref)
	if err != nil {
		return fmt.Errorf("failed to pull: %w", err)
	}

	buf := make([]byte, 4096)
	for {
		n, readErr := reader.Read(buf)
		if readErr == io.EOF && n == 0 {
			// exit as we read all the chunks
			return nil
		}
		if readErr != io.EOF && readErr != nil {
			// return if a non-nil error and stream was not fully read
			return fmt.Errorf("failed to read: %w", err)
		}

		// forward data
		err = stream.Send(&storetypes.Object{
			Name: *ref.Name,
			Type: *ref.Type,
			Size: uint64(n),
			Data: buf[:n],
		})
		if err != nil {
			return fmt.Errorf("failed to send: %w", err)
		}
	}
}

func (s storeController) Lookup(ctx context.Context, req *storetypes.ObjectRef) (*storetypes.ObjectRef, error) {
	if req.GetDigest() == "" {
		return nil, fmt.Errorf("digest is required")
	}

	meta, err := s.store.Lookup(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup: %w", err)
	}

	return meta, nil
}

func (s storeController) Delete(_ context.Context, req *storetypes.ObjectRef) (*emptypb.Empty, error) {
	if req.GetDigest() == "" {
		return nil, fmt.Errorf("digest is required")
	}

	err := s.store.Delete(context.Background(), req)
	if err != nil {
		return nil, fmt.Errorf("failed to delete: %w", err)
	}

	return &emptypb.Empty{}, nil
}
