// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package format_test

import (
	"encoding/json"
	"testing"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/cli/cmd/export/format"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
)

// Minimal OASF record JSON with an A2A module so the translator can extract a card.
const testA2ARecordJSON = `{
  "schema_version": "1.0.0",
  "name": "test-a2a-agent",
  "version": "1.0.0",
  "description": "A test A2A agent",
  "modules": [
    {
      "name": "integration/a2a",
      "data": {
        "card_data": {
          "name": "test-a2a-agent",
          "description": "A test A2A agent",
          "version": "1.0.0",
          "protocolVersions": ["0.2.6"],
          "supportedInterfaces": [
            {
              "url": "https://example.com/a2a",
              "protocolBinding": "HTTP+JSON"
            }
          ],
          "capabilities": {},
          "defaultInputModes": ["text"],
          "defaultOutputModes": ["text"],
          "skills": [
            {
              "id": "test-skill",
              "name": "Test Skill",
              "description": "A skill for testing"
            }
          ]
        },
        "card_schema_version": "v1.0.0"
      }
    }
  ]
}`

func newA2ATestRecord(t *testing.T) *corev1.Record {
	t.Helper()

	var data structpb.Struct

	require.NoError(t, protojson.Unmarshal([]byte(testA2ARecordJSON), &data))

	return &corev1.Record{Data: &data}
}

func TestGetA2AFormatter(t *testing.T) {
	f, err := format.GetFormatter("a2a")
	require.NoError(t, err)
	assert.NotNil(t, f)
}

func TestA2AFormatter_Format(t *testing.T) {
	f, err := format.GetFormatter("a2a")
	require.NoError(t, err)

	t.Run("formats a record with A2A module data", func(t *testing.T) {
		record := newA2ATestRecord(t)

		output, err := f.Format(record)
		require.NoError(t, err)
		assert.NotEmpty(t, output)

		var parsed map[string]any
		require.NoError(t, json.Unmarshal(output, &parsed))
		assert.Equal(t, "test-a2a-agent", parsed["name"])
	})

	t.Run("returns error for record with nil data", func(t *testing.T) {
		record := &corev1.Record{}

		_, err := f.Format(record)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "record contains no data")
	})

	t.Run("returns error for record without A2A module data", func(t *testing.T) {
		record := newTestRecord()

		_, err := f.Format(record)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to translate record to A2A AgentCard")
	})
}

func TestA2AFormatter_FileExtension(t *testing.T) {
	f, err := format.GetFormatter("a2a")
	require.NoError(t, err)
	assert.Equal(t, ".json", f.FileExtension())
}
