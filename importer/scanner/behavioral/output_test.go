// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package behavioral

import (
	"testing"

	"github.com/agntcy/dir/importer/scanner/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseOutput_Empty(t *testing.T) {
	result, err := parseOutput([]byte("[]"))
	require.NoError(t, err)
	assert.True(t, result.Safe)
	assert.Empty(t, result.Findings)
}

func TestParseOutput_Safe(t *testing.T) {
	raw := `[{"tool_name":"my_tool","status":"completed","is_safe":true,"findings":{}}]`

	result, err := parseOutput([]byte(raw))
	require.NoError(t, err)
	assert.True(t, result.Safe)
	assert.Empty(t, result.Findings)
}

func TestParseOutput_UnsafeHighSeverity(t *testing.T) {
	raw := `[
		{
			"tool_name": "execute_system_command",
			"status": "completed",
			"is_safe": false,
			"findings": {
				"behavioral_analyzer": {
					"severity": "HIGH",
					"threat_summary": "PROMPT INJECTION detected",
					"threat_names": ["PROMPT INJECTION"],
					"total_findings": 1
				}
			}
		}
	]`

	result, err := parseOutput([]byte(raw))
	require.NoError(t, err)
	assert.False(t, result.Safe)
	require.Len(t, result.Findings, 1)
	assert.Equal(t, types.SeverityError, result.Findings[0].Severity)
	assert.Contains(t, result.Findings[0].Message, "execute_system_command")
	assert.Contains(t, result.Findings[0].Message, "PROMPT INJECTION")
}

func TestParseOutput_MediumSeverity(t *testing.T) {
	raw := `[
		{
			"tool_name": "some_tool",
			"status": "completed",
			"is_safe": false,
			"findings": {
				"behavioral_analyzer": {
					"severity": "MEDIUM",
					"threat_summary": "suspicious behavior",
					"threat_names": [],
					"total_findings": 1
				}
			}
		}
	]`

	result, err := parseOutput([]byte(raw))
	require.NoError(t, err)
	assert.False(t, result.Safe)
	require.Len(t, result.Findings, 1)
	assert.Equal(t, types.SeverityWarning, result.Findings[0].Severity)
}

func TestParseOutput_WithLeadingLogLines(t *testing.T) {
	raw := `2026-03-10 WARNING - some log line
[{"tool_name":"t","status":"completed","is_safe":true,"findings":{}}]`

	result, err := parseOutput([]byte(raw))
	require.NoError(t, err)
	assert.True(t, result.Safe)
}

func TestParseOutput_InvalidJSON(t *testing.T) {
	_, err := parseOutput([]byte("not json at all"))
	assert.Error(t, err)
}

func TestParseOutput_MultipleTools(t *testing.T) {
	raw := `[
		{
			"tool_name": "tool_a",
			"status": "completed",
			"is_safe": false,
			"findings": {
				"behavioral_analyzer": {
					"severity": "HIGH",
					"threat_summary": "threat A",
					"threat_names": ["INJECTION"],
					"total_findings": 1
				}
			}
		},
		{
			"tool_name": "tool_b",
			"status": "completed",
			"is_safe": true,
			"findings": {}
		},
		{
			"tool_name": "tool_c",
			"status": "completed",
			"is_safe": false,
			"findings": {
				"behavioral_analyzer": {
					"severity": "CRITICAL",
					"threat_summary": "critical threat",
					"threat_names": ["DATA EXFILTRATION"],
					"total_findings": 1
				}
			}
		}
	]`

	result, err := parseOutput([]byte(raw))
	require.NoError(t, err)
	assert.False(t, result.Safe)
	require.Len(t, result.Findings, 2)
}

func TestMapSeverity(t *testing.T) {
	tests := []struct {
		input    string
		expected types.FindingSeverity
	}{
		{"CRITICAL", types.SeverityError},
		{"HIGH", types.SeverityError},
		{"MEDIUM", types.SeverityWarning},
		{"LOW", types.SeverityInfo},
		{"INFO", types.SeverityInfo},
		{"", types.SeverityInfo},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, mapSeverity(tt.input))
		})
	}
}

func TestTrimToJSON(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"already json", `[{"a":1}]`, `[{"a":1}]`},
		{"leading logs", "WARNING log\n[{\"a\":1}]", "[{\"a\":1}]"},
		{"no bracket", "no json here", "no json here"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, string(trimToJSON([]byte(tt.input))))
		})
	}
}
