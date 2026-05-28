// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package exportfmt is the shared registry of OASF record export
// formats. It serves both the `dirctl export` CLI command (which adds
// batch-export behaviour on top via BatchFormatter) and the
// AgentFinder HTTP endpoint GET /v1/agents/{cid}/export, which writes
// the same bytes — byte-identical to the CLI — into the HTTP response
// body with the appropriate Content-Type.
//
// Keeping the registry under api/ (rather than under cli/) makes the
// dependency direction one-way: server → api/exportfmt; cli also
// consumes api/exportfmt and layers on file-system semantics.
package exportfmt

import (
	"errors"
	"fmt"
	"sort"
	"sync"

	corev1 "github.com/agntcy/dir/api/core/v1"
)

// ErrUnsupportedRecord signals that a record cannot be projected into
// the requested export format because it lacks the OASF module that
// format depends on (e.g. agent-skill requires
// "core/language_model/agentskills"; a2a requires "integration/a2a";
// mcp-ghcopilot requires "integration/mcp").
//
// This is a *data* mismatch, not a server fault: the request was
// well-formed, the format is registered, and the record exists — it
// just doesn't carry the bits the format reads. Callers (notably the
// HTTP gateway) should map this to a 4xx response (FailedPrecondition
// / HTTP 400) rather than a 5xx, so this kind of failure doesn't page
// operators when a client asks for an incompatible projection.
//
// Use errors.Is(err, ErrUnsupportedRecord) to detect it; the original
// translator error remains in the chain via Unwrap so its detail is
// preserved for logging.
var ErrUnsupportedRecord = errors.New("record cannot be projected to requested format")

// AsUnsupportedRecord tags err so it satisfies
// errors.Is(err, ErrUnsupportedRecord) without altering err.Error().
// Returns nil if err is nil.
//
// Formatters use this when their translator step rejects a record:
// keep the original message (callers and tests still see the
// translator's wording) while also routing the failure through the
// proper "unsupported record" category.
func AsUnsupportedRecord(err error) error {
	if err == nil {
		return nil
	}

	return &unsupportedRecordError{inner: err}
}

// unsupportedRecordError adapts an arbitrary translator error so it
// matches ErrUnsupportedRecord under errors.Is while preserving the
// underlying message and unwrap chain.
type unsupportedRecordError struct {
	inner error
}

func (e *unsupportedRecordError) Error() string { return e.inner.Error() }
func (e *unsupportedRecordError) Unwrap() error { return e.inner }

// Is reports the wrapper's identity (ErrUnsupportedRecord) without
// consuming the target slot — errors.Is keeps walking the chain so
// callers can still match deeper wrapped errors if they want to.
func (e *unsupportedRecordError) Is(target error) bool {
	return target == ErrUnsupportedRecord
}

// Format-name string constants. Use these instead of literals so the
// CLI flag, controller request validation, and registry init stay in
// sync.
const (
	FormatOASF         = "oasf"
	FormatA2A          = "a2a"
	FormatAgentSkill   = "agent-skill"
	FormatSkillAlias   = "skill" // deprecated alias for FormatAgentSkill
	FormatMCPGHCopilot = "mcp-ghcopilot"
)

// File extensions returned by FileExtension(). Centralised so callers
// that need to choose an HTTP Content-Type can build a stable mapping
// from extension to media type without depending on each formatter's
// implementation details.
const (
	ExtJSON     = ".json"
	ExtMarkdown = ".md"
)

// Formatter converts a single OASF record into a target representation.
//
// Format(record) and FileExtension() are the bytes-and-extension pair
// used by both stdout/file CLI output and the HTTP gateway. The HTTP
// gateway derives a response Content-Type from FileExtension() via
// ContentTypeForExtension; formatters that want a non-default mapping
// should add a case there rather than overriding here, so the mapping
// remains discoverable in one place.
type Formatter interface {
	// Format transforms the OASF record into the target representation.
	// Returns the bytes that would be written to stdout / file / HTTP
	// response body.
	Format(record *corev1.Record) ([]byte, error)

	// FileExtension returns the default file extension for this
	// format (e.g. ".json", ".md"). Used by the CLI for output
	// filenames and by the HTTP gateway to derive a Content-Type.
	FileExtension() string
}

// BatchFormatter extends Formatter for formats that need custom
// multi-record export behaviour (e.g. merging MCP servers into one
// config, or creating per-skill subdirectories). Formatters that do
// not implement BatchFormatter get per-record file writing via
// DefaultBatchExport.
type BatchFormatter interface {
	Formatter

	// FormatBatch exports multiple records to outputDir.
	// When allVersions is true, the version is included in filenames
	// to preserve every version; otherwise only the latest per name
	// is kept. Returns the number of records successfully exported.
	FormatBatch(records []*corev1.Record, outputDir string, allVersions bool) (int, error)
}

var (
	registryMu sync.RWMutex
	formatters = map[string]Formatter{}
)

// RegisterFormatter registers a named formatter. It is safe for
// concurrent use. Called by each formatter's init().
func RegisterFormatter(name string, f Formatter) {
	registryMu.Lock()
	defer registryMu.Unlock()

	formatters[name] = f
}

// GetFormatter returns the formatter registered under name, or an
// error if not found. The error message mentions the supplied name so
// it is suitable to surface directly to CLI users / HTTP clients.
func GetFormatter(name string) (Formatter, error) {
	registryMu.RLock()
	defer registryMu.RUnlock()

	f, ok := formatters[name]
	if !ok {
		return nil, fmt.Errorf("unsupported export format %q", name)
	}

	return f, nil
}

// KnownFormats returns the names of every registered formatter,
// sorted lexicographically. Useful for building CLI help text and
// HTTP error messages that enumerate supported formats. The returned
// slice is a fresh copy and safe for the caller to mutate.
func KnownFormats() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()

	out := make([]string, 0, len(formatters))
	for name := range formatters {
		out = append(out, name)
	}

	sort.Strings(out)

	return out
}

// ContentTypeForExtension returns the HTTP Content-Type to advertise
// for the given file extension. Centralised so the mapping is
// auditable in one place rather than scattered across formatters.
// Returns "application/octet-stream" for unknown extensions — callers
// that care about strict typing should check the value explicitly.
func ContentTypeForExtension(ext string) string {
	switch ext {
	case ExtJSON:
		return "application/json"
	case ExtMarkdown:
		// text/markdown is the IANA-registered media type for
		// CommonMark/Markdown (RFC 7763). Charset is mandatory under
		// the registration.
		return "text/markdown; charset=utf-8"
	default:
		return "application/octet-stream"
	}
}
