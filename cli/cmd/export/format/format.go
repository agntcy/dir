// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package format

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

// BatchFormatter extends Formatter for formats that need custom multi-record
// export behaviour (e.g. merging MCP servers into one config, or creating
// per-skill subdirectories).
// Formatters that do not implement BatchFormatter get per-record file writing
// via DefaultBatchExport.
type BatchFormatter interface {
	Formatter

	// FormatBatch exports multiple records to outputDir.
	// Returns the number of records successfully exported.
	FormatBatch(records []*corev1.Record, outputDir string) (int, error)
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

// RecordName extracts the name field from a record's data.
func RecordName(record *corev1.Record) string {
	data := record.GetData()
	if data == nil {
		return ""
	}

	if nameVal, ok := data.GetFields()["name"]; ok {
		return nameVal.GetStringValue()
	}

	return ""
}

// SanitizeName replaces characters unsafe for filenames with hyphens.
func SanitizeName(name string) string {
	r := strings.NewReplacer("/", "-", "\\", "-", ":", "-", " ", "-")

	return r.Replace(name)
}

// DefaultBatchExport provides per-record file writing for formatters that
// do not implement BatchFormatter. Each record is written as
// <outputDir>/<sanitised-name><ext>.
func DefaultBatchExport(f Formatter, records []*corev1.Record, outputDir string) (int, error) {
	exported := 0

	for i, record := range records {
		name := RecordName(record)
		if name == "" {
			name = fmt.Sprintf("record_%d", i)
		}

		output, err := f.Format(record)
		if err != nil {
			return exported, fmt.Errorf("failed to format record %q: %w", name, err)
		}

		outPath := filepath.Join(outputDir, SanitizeName(name)+f.FileExtension())

		if err := os.WriteFile(outPath, output, 0o600); err != nil { //nolint:mnd
			return exported, fmt.Errorf("failed to write %s: %w", outPath, err)
		}

		exported++
	}

	return exported, nil
}
