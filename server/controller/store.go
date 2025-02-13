// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"fmt"
	coretypes "github.com/agntcy/dir/api/core/v1alpha1"
	"io"
	"log"

	storetypes "github.com/agntcy/dir/api/store/v1alpha1"
	"github.com/agntcy/dir/server/types"

	"google.golang.org/protobuf/types/known/emptypb"
)

type storeController struct {
	store types.StoreService
	storetypes.UnimplementedStoreServiceServer
}

func NewStoreController(store types.StoreService) storetypes.StoreServiceServer {
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

	metadata := firstMessage.GetMetadata()
	if metadata == nil {
		return fmt.Errorf("metadata is required")
	}

	log.Printf("Received metadata: Type=%v, Name=%s, Annotations=%v\n", metadata.Type, metadata.Name, metadata.Annotations)

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

	digest, err := s.store.Push(
		context.Background(),
		&coretypes.ObjectMeta{
			Type:        metadata.Type,
			Name:        metadata.Name,
			Annotations: metadata.Annotations,
			Digest:      metadata.Digest,
		},
		pr,
	)
	if err != nil {
		return fmt.Errorf("failed to push: %w", err)
	}

	return stream.SendAndClose(&coretypes.ObjectRef{Digest: digest})
}

func (s storeController) Pull(req *coretypes.ObjectRef, stream storetypes.StoreService_PullServer) error {
	if req.GetDigest() == nil || len(req.GetDigest().GetValue()) == 0 {
		return fmt.Errorf("digest is required")
	}

	reader, err := s.store.Pull(context.Background(), req.Digest)
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
		err = stream.Send(&coretypes.Object{Data: buf[:n]})
		if err != nil {
			return fmt.Errorf("failed to send: %w", err)
		}
	}
}

func (s storeController) Lookup(ctx context.Context, req *coretypes.ObjectRef) (*coretypes.ObjectMeta, error) {
	if req.GetDigest() == nil || len(req.GetDigest().GetValue()) == 0 {
		return nil, fmt.Errorf("digest is required")
	}

	meta, err := s.store.Lookup(ctx, req.Digest)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup: %w", err)
	}

	return meta, nil
}

func (s storeController) Delete(_ context.Context, req *coretypes.ObjectRef) (*emptypb.Empty, error) {
	if req.GetDigest() == nil || len(req.GetDigest().GetValue()) == 0 {
		return nil, fmt.Errorf("digest is required")
	}

	err := s.store.Delete(context.Background(), req.Digest)
	if err != nil {
		return nil, fmt.Errorf("failed to delete: %w", err)
	}

	return &emptypb.Empty{}, nil
}
