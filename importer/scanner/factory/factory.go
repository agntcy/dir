// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package factory

import (
	"fmt"
	"sort"
	"sync"

	scannerconfig "github.com/agntcy/dir/importer/scanner/config"
	"github.com/agntcy/dir/importer/scanner/types"
)

// ScannerFactory creates a Scanner from the shared scanner config.
type ScannerFactory func(cfg scannerconfig.Config) types.Scanner

var (
	registry = make(map[string]ScannerFactory)
	mu       sync.RWMutex
)

// Register registers a ScannerFactory under the given name.
// It panics if the same name is registered twice to prevent duplications at compile-time.
// Scanner implementations should call this from their init() function.
func Register(name string, factory ScannerFactory) {
	mu.Lock()
	defer mu.Unlock()

	if _, exists := registry[name]; exists {
		panic(fmt.Sprintf("scanner: already registered for name: %s", name))
	}

	registry[name] = factory
}

// NewScanners creates Scanner instances for all registered scanners.
func NewScanners(cfg scannerconfig.Config) ([]types.Scanner, error) {
	mu.RLock()
	defer mu.RUnlock()

	scanners := make([]types.Scanner, 0, len(registry))

	for _, name := range sortedRegistryNames() {
		scanners = append(scanners, registry[name](cfg))
	}

	return scanners, nil
}

func sortedRegistryNames() []string {
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}

	sort.Strings(names)

	return names
}
