// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package format

import (
	"fmt"
	"sync"

	corev1 "github.com/agntcy/dir/api/core/v1"
)

// Formatter converts an OASF record into a target format.
type Formatter interface {
	// Format transforms the OASF record into the target representation.
	Format(record *corev1.Record) ([]byte, error)

	// FileExtension returns the default file extension for this format (e.g. ".json", ".md").
	FileExtension() string
}

const ExtJSON = ".json"

var (
	registryMu sync.RWMutex
	formatters = map[string]Formatter{}
)

// RegisterFormatter registers a named formatter. It is safe for concurrent use.
func RegisterFormatter(name string, f Formatter) {
	registryMu.Lock()
	defer registryMu.Unlock()

	formatters[name] = f
}

// GetFormatter returns the formatter registered under name, or an error if not found.
func GetFormatter(name string) (Formatter, error) {
	registryMu.RLock()
	defer registryMu.RUnlock()

	f, ok := formatters[name]
	if !ok {
		return nil, fmt.Errorf("unsupported export format %q", name)
	}

	return f, nil
}
