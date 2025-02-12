// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package types

import (
	"context"
	"io"

	coretypes "github.com/agntcy/dir/api/core/v1alpha1"
	registrytypes "github.com/agntcy/dir/api/registry/v1alpha1"
)

// StoreService handles management of content-addressable object storage
type StoreService interface {
	Push(ctx context.Context, meta *registrytypes.ObjectMeta, contents io.Reader) (*coretypes.Digest, error)
	Pull(ctx context.Context, ref *coretypes.Digest) (io.Reader, error)
	Lookup(ctx context.Context, ref *coretypes.Digest) (*registrytypes.ObjectMeta, error)
	Delete(ctx context.Context, ref *coretypes.Digest) error
}

// PublishService handles management of mutable references to immutable objects
type PublishService interface {
	Publish(ctx context.Context, tag string, ref *coretypes.Digest) error
	Unpublish(ctx context.Context, tag string) error
	Resolve(ctx context.Context, tag string) (*coretypes.Digest, error)
}

// Registry handles all operations around content
// management and logic.
type Registry interface {
	Store() StoreService
	Publish() PublishService
}

type RegistryOptions struct {
	store   StoreService
	publish PublishService
}

type RegistryOption func(*RegistryOptions) error

func WithStoreService(store StoreService) RegistryOption {
	return func(options *RegistryOptions) error {
		options.store = store
		return nil
	}
}

func WithPublishService(publish PublishService) RegistryOption {
	return func(options *RegistryOptions) error {
		options.publish = publish
		return nil
	}
}

type registry struct {
	options *RegistryOptions
}

func NewRegistry(opts ...RegistryOption) (Registry, error) {
	options := &RegistryOptions{}
	for _, opt := range opts {
		if err := opt(options); err != nil {
			return nil, err
		}
	}

	return &registry{
		options: options,
	}, nil
}

func (r registry) Store() StoreService {
	return r.options.store
}

func (r registry) Publish() PublishService {
	return r.options.publish
}
