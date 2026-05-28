// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint
package exportfmt_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	oasfv1alpha1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/agntcy/oasf/types/v1alpha1"
	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/api/exportfmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestRecord() *corev1.Record {
	return corev1.New(&oasfv1alpha1.Record{
		Name:          "test-agent",
		SchemaVersion: "v0.5.0",
		Description:   "A test agent for export formatting",
		Version:       "1.0.0",
	})
}

func TestGetFormatter(t *testing.T) {
	t.Run("returns oasf formatter", func(t *testing.T) {
		f, err := exportfmt.GetFormatter("oasf")
		require.NoError(t, err)
		assert.NotNil(t, f)
	})

	t.Run("returns error for unknown format", func(t *testing.T) {
		f, err := exportfmt.GetFormatter("nonexistent")
		assert.Nil(t, f)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported export format")
	})
}

func TestOASFFormatter_Format(t *testing.T) {
	f, err := exportfmt.GetFormatter("oasf")
	require.NoError(t, err)

	t.Run("formats a valid record as JSON", func(t *testing.T) {
		record := newTestRecord()

		output, err := f.Format(record)
		require.NoError(t, err)
		assert.NotEmpty(t, output)

		var parsed map[string]any
		require.NoError(t, json.Unmarshal(output, &parsed))
		assert.Equal(t, "test-agent", parsed["name"])
		assert.Equal(t, "A test agent for export formatting", parsed["description"])
	})

	t.Run("returns error for record with nil data", func(t *testing.T) {
		record := &corev1.Record{}

		_, err := f.Format(record)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "record contains no data")
	})
}

func TestOASFFormatter_FileExtension(t *testing.T) {
	f, err := exportfmt.GetFormatter("oasf")
	require.NoError(t, err)
	assert.Equal(t, ".json", f.FileExtension())
}

// TestAsUnsupportedRecord pins down the contract that controllers
// (notably the HTTP gateway) depend on: AsUnsupportedRecord preserves
// the original error message and wrap chain while making the result
// match ErrUnsupportedRecord under errors.Is. If this drifts, the
// AgentFinder ExportAgent RPC will silently return 500 for what
// should be 400 responses, so we lock the behaviour in here.
func TestAsUnsupportedRecord(t *testing.T) {
	t.Run("nil in -> nil out", func(t *testing.T) {
		assert.NoError(t, exportfmt.AsUnsupportedRecord(nil))
	})

	t.Run("matches ErrUnsupportedRecord under errors.Is", func(t *testing.T) {
		inner := errors.New("translator: missing integration/a2a module")
		wrapped := exportfmt.AsUnsupportedRecord(inner)

		require.Error(t, wrapped)
		assert.ErrorIs(t, wrapped, exportfmt.ErrUnsupportedRecord)
	})

	t.Run("preserves the original error message verbatim", func(t *testing.T) {
		inner := errors.New("translator: missing integration/a2a module")
		wrapped := exportfmt.AsUnsupportedRecord(inner)

		// We deliberately do NOT prepend "unsupported record:" or
		// similar — formatters already supply a human-readable
		// translator message ("failed to translate record to A2A
		// AgentCard: ...") and the controller produces its own
		// preamble. Doubling the prefix would noise up the gRPC
		// status message clients see.
		assert.Equal(t, inner.Error(), wrapped.Error())
	})

	t.Run("unwrap chain still reaches the inner error", func(t *testing.T) {
		inner := errors.New("translator: missing integration/a2a module")
		wrapped := exportfmt.AsUnsupportedRecord(fmt.Errorf("failed to translate: %w", inner))

		// Walking the chain must surface both the sentinel and the
		// original cause so callers can keep using errors.Is to
		// distinguish e.g. translator-specific failures.
		assert.ErrorIs(t, wrapped, exportfmt.ErrUnsupportedRecord)
		assert.ErrorIs(t, wrapped, inner)
	})
}
