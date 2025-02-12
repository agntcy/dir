// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package oci

import (
	"context"

	coretypes "github.com/agntcy/dir/api/core/v1alpha1"
	"github.com/agntcy/dir/registry/types"
)

type publish struct {
}

func NewPublishService() (types.PublishService, error) {
	return &publish{}, nil
}

func (t *publish) Publish(ctx context.Context, tag string, ref *coretypes.Digest) error {
	panic("not implemented")
}

func (t *publish) Unpublish(ctx context.Context, tag string) error {
	panic("not implemented")
}

func (t *publish) Resolve(ctx context.Context, tag string) (*coretypes.Digest, error) {
	panic("not implemented")
}
