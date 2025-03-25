// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:wrapcheck
package controller

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"

	coretypes "github.com/agntcy/dir/api/core/v1alpha1"
	storetypes "github.com/agntcy/dir/api/store/v1alpha1"
	"github.com/agntcy/dir/server/types"
	"google.golang.org/protobuf/types/known/emptypb"
)

type storeCtrl struct {
	storetypes.UnimplementedStoreServiceServer
	store types.StoreAPI
}

func NewStoreController(store types.StoreAPI) storetypes.StoreServiceServer {
	return &storeCtrl{
		UnimplementedStoreServiceServer: storetypes.UnimplementedStoreServiceServer{},
		store:                           store,
	}
}

func (s storeCtrl) Push(stream storetypes.StoreService_PushServer) error {
	// TODO: validate
	firstMessage, err := stream.Recv()
	if err != nil {
		return fmt.Errorf("failed to receive first message: %w", err)
	}

	log.Printf("Pushing object %s with digest %v\n", firstMessage.GetRef().GetType(), firstMessage.GetRef().GetDigest())

	// lookup (skip if exists)
	ref, err := s.store.Lookup(stream.Context(), firstMessage.GetRef())
	if err == nil {
		return stream.SendAndClose(ref)
	}

	// read packets into a pipe in the background
	pr, pw := io.Pipe()
	go func() {
		defer pw.Close()

		if len(firstMessage.GetData()) > 0 {
			if _, err := pw.Write(firstMessage.GetData()); err != nil {
				pw.CloseWithError(err)

				return
			}
		}

		for {
			obj, err := stream.Recv()
			if errors.Is(err, io.EOF) {
				return
			}

			if err != nil {
				pw.CloseWithError(err)

				return
			}

			if _, err := pw.Write(obj.GetData()); err != nil {
				pw.CloseWithError(err)

				return
			}
		}
	}()

	// push
	ref, err = s.store.Push(stream.Context(), firstMessage.GetRef(), pr)
	if err != nil {
		return fmt.Errorf("failed to push: %w", err)
	}

	return stream.SendAndClose(ref)
}

func (s storeCtrl) Pull(req *coretypes.ObjectRef, stream storetypes.StoreService_PullServer) error {
	// lookup (maybe not needed)
	ref, err := s.store.Lookup(stream.Context(), req)
	if err != nil {
		return fmt.Errorf("failed to lookup: %w", err)
	}

	// pull
	reader, err := s.store.Pull(stream.Context(), ref)
	if err != nil {
		return fmt.Errorf("failed to pull: %w", err)
	}

	buf := make([]byte, 4096) //nolint:mnd

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
		err = stream.Send(&coretypes.Object{
			Data: buf[:n],
		})
		if err != nil {
			return fmt.Errorf("failed to send: %w", err)
		}
	}
}

func (s storeCtrl) Lookup(ctx context.Context, req *coretypes.ObjectRef) (*coretypes.ObjectRef, error) {
	// validate
	if req.GetDigest() == "" {
		return nil, errors.New("digest is required")
	}

	// TODO: add caching to avoid querying the Storage API

	// lookup
	meta, err := s.store.Lookup(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup: %w", err)
	}

	return meta, nil
}

func (s storeCtrl) Delete(ctx context.Context, req *coretypes.ObjectRef) (*emptypb.Empty, error) {
	if req.GetDigest() == "" {
		return nil, errors.New("digest is required")
	}

	err := s.store.Delete(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to delete: %w", err)
	}

	return &emptypb.Empty{}, nil
}
