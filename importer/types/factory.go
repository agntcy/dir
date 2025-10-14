// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package types

import "fmt"

// Factory creates Importer instances based on registry type.
type Factory struct {
	importers map[RegistryType]func(config ImportConfig) (Importer, error)
}

// NewFactory creates a new importer factory.
func NewFactory() *Factory {
	return &Factory{
		importers: make(map[RegistryType]func(config ImportConfig) (Importer, error)),
	}
}

// Register registers a constructor function for a given registry type.
func (f *Factory) Register(registryType RegistryType, constructor func(ImportConfig) (Importer, error)) {
	f.importers[registryType] = constructor
}

// Create creates a new Importer instance for the given configuration.
func (f *Factory) Create(config ImportConfig) (Importer, error) {
	constructor, exists := f.importers[config.RegistryType]
	if !exists {
		return nil, fmt.Errorf("unsupported registry type: %s", config.RegistryType)
	}

	return constructor(config)
}
