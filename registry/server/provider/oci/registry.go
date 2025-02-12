// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package oci

import (
	"github.com/agntcy/dir/registry/types"
	"github.com/spf13/afero"
)

var DefaultFs = afero.NewOsFs()

func NewRegistryProvider(config Config) (types.Registry, error) {
	store, err := NewStoreService(config)
	if err != nil {
		return nil, err
	}

	publish, err := NewPublishService()
	if err != nil {
		return nil, err
	}

	registry, err := types.NewRegistry(
		types.WithStoreService(store),
		types.WithPublishService(publish),
	)
	if err != nil {
		return nil, err
	}

	return registry, nil
}
