// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package types

import (
	"context"
	"io"

	coretypes "github.com/agntcy/dir/api/core/v1alpha1"
)

// StoreService handles management of content-addressable object storage.
type StoreService interface {
	Push(ctx context.Context, meta *coretypes.ObjectMeta, contents io.Reader) (*coretypes.Digest, error)
	Pull(ctx context.Context, ref *coretypes.Digest) (io.Reader, error)
	Lookup(ctx context.Context, ref *coretypes.Digest) (*coretypes.ObjectMeta, error)
	Delete(ctx context.Context, ref *coretypes.Digest) error
}

// PublishService handles management of network publication and retrieval of records.
type PublishService interface {
	Publish(ctx context.Context, record string, ref *coretypes.Digest) error
	Unpublish(ctx context.Context, record string) error
	Resolve(ctx context.Context, record string) (*coretypes.Digest, error)
}
