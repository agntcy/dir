// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:testifylint
package corev1

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectOASFVersion(t *testing.T) {
	tests := []struct {
		name        string
		jsonData    string
		expectedVer string
		expectError bool
	}{
		{
			name:        "v0.3.1 with schema_version",
			jsonData:    `{"schema_version": "v0.3.1", "name": "test-agent"}`,
			expectedVer: "v0.3.1",
			expectError: false,
		},
		{
			name:        "v0.4.0 with schema_version",
			jsonData:    `{"schema_version": "v0.4.0", "name": "test-agent"}`,
			expectedVer: "v0.4.0",
			expectError: false,
		},
		{
			name:        "v0.5.0 with schema_version",
			jsonData:    `{"schema_version": "v0.5.0", "name": "test-record"}`,
			expectedVer: "v0.5.0",
			expectError: false,
		},
		{
			name:        "no schema_version defaults to v0.3.1",
			jsonData:    `{"name": "test-agent", "version": "1.0"}`,
			expectedVer: "v0.3.1",
			expectError: false,
		},
		{
			name:        "invalid json",
			jsonData:    `{"name": "test-agent"`,
			expectedVer: "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			version, err := DetectOASFVersion([]byte(tt.jsonData))

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedVer, version)
			}
		})
	}
}

func TestLoadOASFFromReader(t *testing.T) {
	tests := []struct {
		name        string
		jsonData    string
		expectError bool
		expectV1    bool
		expectV2    bool
		expectV3    bool
	}{
		{
			name:        "valid v0.3.1 agent",
			jsonData:    `{"schema_version": "v0.3.1", "name": "test-agent", "version": "1.0"}`,
			expectError: false,
			expectV1:    true,
		},
		{
			name:        "valid v0.4.0 agent record",
			jsonData:    `{"schema_version": "v0.4.0", "name": "test-agent", "version": "1.0"}`,
			expectError: false,
			expectV2:    true,
		},
		{
			name:        "valid v0.5.0 record",
			jsonData:    `{"schema_version": "v0.5.0", "name": "test-record", "version": "1.0"}`,
			expectError: false,
			expectV3:    true,
		},
		{
			name:        "no schema_version defaults to v0.3.1",
			jsonData:    `{"name": "test-agent", "version": "1.0"}`,
			expectError: false,
			expectV1:    true,
		},
		{
			name:        "unsupported version",
			jsonData:    `{"schema_version": "v0.6.0", "name": "test-record"}`,
			expectError: true,
		},
		{
			name:        "invalid json",
			jsonData:    `{"name": "test-agent"`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.jsonData)
			record, err := LoadOASFFromReader(reader)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, record)
			} else {
				require.NoError(t, err)
				require.NotNil(t, record)

				//nolint:gocritic // if-else chain is clearer than switch for boolean flag testing in tests
				if tt.expectV1 {
					assert.NotNil(t, record.GetV1())
					assert.Nil(t, record.GetV2())
					assert.Nil(t, record.GetV3())
				} else if tt.expectV2 {
					assert.Nil(t, record.GetV1())
					assert.NotNil(t, record.GetV2())
					assert.Nil(t, record.GetV3())
				} else if tt.expectV3 {
					assert.Nil(t, record.GetV1())
					assert.Nil(t, record.GetV2())
					assert.NotNil(t, record.GetV3())
				}
			}
		})
	}
}

func TestLoadOASFFromBytes(t *testing.T) {
	tests := []struct {
		name        string
		jsonData    string
		expectError bool
		expectV1    bool
		expectV2    bool
		expectV3    bool
	}{
		{
			name:        "valid v0.3.1 agent",
			jsonData:    `{"schema_version": "v0.3.1", "name": "test-agent", "version": "1.0"}`,
			expectError: false,
			expectV1:    true,
		},
		{
			name:        "valid v0.4.0 agent record",
			jsonData:    `{"schema_version": "v0.4.0", "name": "test-agent", "version": "1.0"}`,
			expectError: false,
			expectV2:    true,
		},
		{
			name:        "valid v0.5.0 record",
			jsonData:    `{"schema_version": "v0.5.0", "name": "test-record", "version": "1.0"}`,
			expectError: false,
			expectV3:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			record, err := LoadOASFFromBytes([]byte(tt.jsonData))

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, record)
			} else {
				require.NoError(t, err)
				require.NotNil(t, record)

				//nolint:gocritic // if-else chain is clearer than switch for boolean flag testing in tests
				if tt.expectV1 {
					assert.NotNil(t, record.GetV1())
					assert.Nil(t, record.GetV2())
					assert.Nil(t, record.GetV3())
				} else if tt.expectV2 {
					assert.Nil(t, record.GetV1())
					assert.NotNil(t, record.GetV2())
					assert.Nil(t, record.GetV3())
				} else if tt.expectV3 {
					assert.Nil(t, record.GetV1())
					assert.Nil(t, record.GetV2())
					assert.NotNil(t, record.GetV3())
				}
			}
		})
	}
}
