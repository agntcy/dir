// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package exportfmt

import (
	"errors"
	"fmt"
	"sort"
	"sync"

	corev1 "github.com/agntcy/dir/api/core/v1"
)

// ErrUnsupportedRecord indicates the record lacks the OASF module required by
// the requested format (e.g. asking for "a2a" on a record without
// integration/a2a). The request was well-formed; the data simply doesn't
// carry what the format reads.
var ErrUnsupportedRecord = errors.New("record does not contain the required module for this format")

// Formatter converts an OASF record into a target format.
type Formatter interface {
	// Format transforms the OASF record into the target representation.
	Format(record *corev1.Record) ([]byte, error)

	// FileExtension returns the default file extension for this format (e.g. ".json", ".md").
	FileExtension() string
}

const (
	ExtJSON  = ".json"
	ExtMD    = ".md"
	ExtTarGz = ".tar.gz"

	FormatOASF             = "oasf"
	FormatA2A              = "a2a"
	FormatAgentSkill       = "agent-skill"
	FormatAgentSkillBundle = "agent-skill-bundle"
	FormatSkill            = "skill"
	FormatMCPGHCopiot      = "mcp-ghcopilot"
)

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

// KnownFormats returns a sorted list of all registered format names.
func KnownFormats() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()

	names := make([]string, 0, len(formatters))
	for name := range formatters {
		names = append(names, name)
	}

	sort.Strings(names)

	return names
}

// ContentTypeForExtension returns the MIME content type for a file extension.
func ContentTypeForExtension(ext string) string {
	switch ext {
	case ExtJSON:
		return "application/json"
	case ExtMD:
		return "text/markdown"
	case ExtTarGz:
		return "application/gzip"
	default:
		return "application/octet-stream"
	}
}
