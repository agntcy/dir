// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package factory

import (
	"context"
	"fmt"
	"sync"

	"github.com/agntcy/dir/importer"
	"github.com/agntcy/dir/importer/config"
	"github.com/agntcy/dir/importer/types"
)

func init() {
	Register(config.RegistryTypeMCP, importer.New)
	Register(config.RegistryTypeFile, importer.New)
}

// ImporterFunc is a function that creates an Importer instance.
type ImporterFunc func(ctx context.Context, client config.ClientInterface, cfg config.Config) (types.Importer, error)

var (
	importers = make(map[config.RegistryType]ImporterFunc)
	mu        sync.RWMutex
)

// Register registers a function that creates an Importer instance for a given registry type.
// It panics if the same registry type is registered twice to prevent duplications at compile-time.
func Register(registryType config.RegistryType, fn ImporterFunc) {
	mu.Lock()
	defer mu.Unlock()

	if _, exists := importers[registryType]; exists {
		panic(fmt.Sprintf("importer already registered for registry type: %s", registryType))
	}

	importers[registryType] = fn
}

// Create creates a new Importer instance for the given client and configuration.
func Create(ctx context.Context, client config.ClientInterface, cfg config.Config) (types.Importer, error) {
	mu.RLock()

	constructor, exists := importers[cfg.RegistryType]

	mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("unsupported registry type: %s", cfg.RegistryType)
	}

	return constructor(ctx, client, cfg)
}
