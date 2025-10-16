// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package types

import "fmt"

// ImporterFunc is a function that creates an Importer instance.
type ImporterFunc func(config ImportConfig) (Importer, error)

// Factory creates Importer instances based on registry type.
type Factory struct {
	importers map[RegistryType]ImporterFunc
}

// NewFactory creates a new importer factory.
func NewFactory() *Factory {
	return &Factory{
		importers: make(map[RegistryType]ImporterFunc),
	}
}

// Register registers a function that creates an Importer instance for a given registry type.
func (f *Factory) Register(registryType RegistryType, fn ImporterFunc) {
	f.importers[registryType] = fn
}

// Create creates a new Importer instance for the given configuration.
func (f *Factory) Create(config ImportConfig) (Importer, error) {
	constructor, exists := f.importers[config.RegistryType]
	if !exists {
		return nil, fmt.Errorf("unsupported registry type: %s", config.RegistryType)
	}

	return constructor(config)
}

// RegisteredTypes returns a list of all registered registry types.
func (f *Factory) RegisteredTypes() []RegistryType {
	types := make([]RegistryType, 0, len(f.importers))
	for t := range f.importers {
		types = append(types, t)
	}

	return types
}

// IsRegistered checks if a registry type is registered.
func (f *Factory) IsRegistered(registryType RegistryType) bool {
	_, exists := f.importers[registryType]

	return exists
}
