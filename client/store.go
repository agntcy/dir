// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"

	coretypes "github.com/agntcy/dir/api/core/v1alpha1"
)

// Push
// TODO: when pushing extension data, squash items and push individually before pushing agent.
// TODO: this allows larger agent file while keeping data in extension compact.
func (c *Client) Push(ctx context.Context, ref *coretypes.ObjectRef, reader io.Reader) (*coretypes.ObjectRef, error) {
	stream, err := c.StoreServiceClient.Push(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create push stream: %w", err)
	}

	buf := make([]byte, chunkSize)

	for {
		n, err := reader.Read(buf)
		if err != nil && err != io.EOF {
			return nil, fmt.Errorf("failed to read data: %w", err)
		}

		if n == 0 {
			break
		}

		obj := &coretypes.Object{
			Ref:  ref,
			Data: buf[:n],
		}

		if err := stream.Send(obj); err != nil {
			return nil, fmt.Errorf("could not send object: %w", err)
		}
	}

	resp, err := stream.CloseAndRecv()
	if err != nil {
		return nil, fmt.Errorf("could not receive response: %w", err)
	}

	return resp, nil
}

// Pull
// TODO: needs to pull each extension data individually and send back.
func (c *Client) Pull(ctx context.Context, ref *coretypes.ObjectRef) (io.Reader, error) {
	stream, err := c.StoreServiceClient.Pull(ctx, ref)
	if err != nil {
		return nil, fmt.Errorf("failed to create pull stream: %w", err)
	}

	var buffer bytes.Buffer

	for {
		obj, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			return nil, fmt.Errorf("failed to receive object: %w", err)
		}

		if _, err := buffer.Write(obj.GetData()); err != nil {
			return nil, fmt.Errorf("failed to write data to buffer: %w", err)
		}
	}

	return &buffer, nil
}

func (c *Client) Lookup(ctx context.Context, ref *coretypes.ObjectRef) (*coretypes.ObjectRef, error) {
	meta, err := c.StoreServiceClient.Lookup(ctx, ref)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup object: %w", err)
	}

	return meta, nil
}
