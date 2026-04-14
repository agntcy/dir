// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package format_test

import (
	"encoding/json"
	"testing"

	oasfv1alpha1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/agntcy/oasf/types/v1alpha1"
	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/cli/cmd/export/format"
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
		f, err := format.GetFormatter("oasf")
		require.NoError(t, err)
		assert.NotNil(t, f)
	})

	t.Run("returns error for unknown format", func(t *testing.T) {
		f, err := format.GetFormatter("nonexistent")
		assert.Nil(t, f)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported export format")
	})
}

func TestOASFFormatter_Format(t *testing.T) {
	f, err := format.GetFormatter("oasf")
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
	f, err := format.GetFormatter("oasf")
	require.NoError(t, err)
	assert.Equal(t, ".json", f.FileExtension())
}
