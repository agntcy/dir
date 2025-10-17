// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package types

import (
	"fmt"

	"github.com/agntcy/dir/importer/config"
)

// ImporterFunc is a function that creates an Importer instance.
type ImporterFunc func(client config.ClientInterface, cfg config.Config) (Importer, error)

// Factory creates Importer instances based on registry type.
type Factory struct {
	importers map[config.RegistryType]ImporterFunc
}

// NewFactory creates a new importer factory.
func NewFactory() *Factory {
	return &Factory{
		importers: make(map[config.RegistryType]ImporterFunc),
	}
}

// Register registers a function that creates an Importer instance for a given registry type.
func (f *Factory) Register(registryType config.RegistryType, fn ImporterFunc) {
	f.importers[registryType] = fn
}

// Create creates a new Importer instance for the given client and configuration.
func (f *Factory) Create(client config.ClientInterface, cfg config.Config) (Importer, error) {
	constructor, exists := f.importers[cfg.RegistryType]
	if !exists {
		return nil, fmt.Errorf("unsupported registry type: %s", cfg.RegistryType)
	}

	return constructor(client, cfg)
}

// RegisteredTypes returns a list of all registered registry types.
func (f *Factory) RegisteredTypes() []config.RegistryType {
	types := make([]config.RegistryType, 0, len(f.importers))
	for t := range f.importers {
		types = append(types, t)
	}

	return types
}

// IsRegistered checks if a registry type is registered.
func (f *Factory) IsRegistered(registryType config.RegistryType) bool {
	_, exists := f.importers[registryType]

	return exists
}
