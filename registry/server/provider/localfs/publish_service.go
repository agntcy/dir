// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package localfs

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"

	coretypes "github.com/agntcy/dir/api/core/v1alpha1"
	"github.com/agntcy/dir/registry/types"
	"github.com/spf13/afero"
)

type publish struct {
	fs afero.Fs
}

func NewPublishService(baseDir string) (types.PublishService, error) {
	dir := filepath.Join(baseDir, "publish")
	if err := DefaultFs.MkdirAll(dir, 0o777); err != nil {
		return nil, err
	}

	return &publish{
		fs: afero.NewBasePathFs(DefaultFs, dir),
	}, nil
}

func (t *publish) Publish(ctx context.Context, tag string, ref *coretypes.Digest) error {
	err := afero.WriteReader(t.fs, tag, bytes.NewReader([]byte(ref.ToString())))
	if err != nil {
		return fmt.Errorf("failed to create tag: %w", err)
	}

	return nil
}

func (p *publish) Unpublish(ctx context.Context, tag string) error {
	err := p.fs.Remove(tag)
	if err != nil {
		return fmt.Errorf("failed to remove tag: %w", err)
	}

	return nil
}

func (p *publish) Resolve(ctx context.Context, tag string) (*coretypes.Digest, error) {
	// Read tag
	digestRaw, err := afero.ReadFile(p.fs, tag)
	if err != nil {
		return &coretypes.Digest{}, fmt.Errorf("failed to get tag: %w", err)
	}

	var digest coretypes.Digest
	if err := digest.FromString(string(digestRaw)); err != nil {
		return &coretypes.Digest{}, fmt.Errorf("failed to parse digest: %w", err)
	}

	return &digest, nil
}
