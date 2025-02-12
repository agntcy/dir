// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"fmt"
	"io"
	"log"

	registrytypes "github.com/agntcy/dir/api/registry/v1alpha1"
	"github.com/agntcy/dir/registry/types"

	"google.golang.org/protobuf/types/known/emptypb"
)

type storeController struct {
	provider types.Registry
	registrytypes.UnimplementedStoreServiceServer
}

func NewStoreController(provider types.Registry) *storeController {
	return &storeController{
		provider:                        provider,
		UnimplementedStoreServiceServer: registrytypes.UnimplementedStoreServiceServer{},
	}
}

func (s storeController) Push(stream registrytypes.StoreService_PushServer) error {
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

	digest, err := s.provider.Store().Push(
		context.Background(),
		&registrytypes.ObjectMeta{
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

	return stream.SendAndClose(&registrytypes.ObjectRef{Digest: digest})
}

func (s storeController) Pull(req *registrytypes.ObjectRef, stream registrytypes.StoreService_PullServer) error {
	if req.GetDigest() == nil || len(req.GetDigest().GetValue()) == 0 {
		return fmt.Errorf("digest is required")
	}

	reader, err := s.provider.Store().Pull(context.Background(), req.Digest)
	if err != nil {
		return fmt.Errorf("failed to pull: %w", err)
	}

	buf := make([]byte, 4096)
	for {
		n, err := reader.Read(buf)
		if err == io.EOF {
			return nil
		}

		if err != nil {
			return fmt.Errorf("failed to read: %w", err)
		}

		err = stream.Send(&registrytypes.Object{Data: buf[:n]})
		if err != nil {
			return fmt.Errorf("failed to send: %w", err)
		}
	}
}

func (s storeController) Lookup(ctx context.Context, req *registrytypes.ObjectRef) (*registrytypes.ObjectMeta, error) {
	if req.GetDigest() == nil || len(req.GetDigest().GetValue()) == 0 {
		return nil, fmt.Errorf("digest is required")
	}

	meta, err := s.provider.Store().Lookup(ctx, req.Digest)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup: %w", err)
	}

	return meta, nil
}

func (s storeController) Delete(_ context.Context, req *registrytypes.ObjectRef) (*emptypb.Empty, error) {
	if req.GetDigest() == nil || len(req.GetDigest().GetValue()) == 0 {
		return nil, fmt.Errorf("digest is required")
	}

	err := s.provider.Store().Delete(context.Background(), req.Digest)
	if err != nil {
		return nil, fmt.Errorf("failed to delete: %w", err)
	}

	return &emptypb.Empty{}, nil
}
