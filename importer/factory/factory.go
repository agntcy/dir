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
	Register(config.ImportTypeMCPRegistry, importer.New)
	Register(config.ImportTypeMCP, importer.New)
	Register(config.ImportTypeA2A, importer.New)
	Register(config.ImportTypeAgentSkill, importer.New)
}

// ImporterFunc is a function that creates an Importer instance.
type ImporterFunc func(ctx context.Context, client config.ClientInterface, cfg config.Config) (types.Importer, error)

var (
	importers = make(map[config.ImportType]ImporterFunc)
	mu        sync.RWMutex
)

// Register registers a function that creates an Importer instance for a given import type.
// It panics if the same import type is registered twice to prevent duplications at compile-time.
func Register(importType config.ImportType, fn ImporterFunc) {
	mu.Lock()
	defer mu.Unlock()

	if _, exists := importers[importType]; exists {
		panic(fmt.Sprintf("importer already registered for import type: %s", importType))
	}

	importers[importType] = fn
}

// Create creates a new Importer instance for the given client and configuration.
func Create(ctx context.Context, client config.ClientInterface, cfg config.Config) (types.Importer, error) {
	mu.RLock()

	constructor, exists := importers[cfg.Type]

	mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("unsupported import type: %s", cfg.Type)
	}

	return constructor(ctx, client, cfg)
}
