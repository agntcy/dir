// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package doctor

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/agntcy/dir/cli/presenter"
	"github.com/agntcy/dir/client"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewOutputDoesNotExposeAuthToken(t *testing.T) {
	output := newOutput(&client.Config{
		ServerAddress: "directory.example.com:443",
		AuthMode:      "oidc",
		AuthToken:     "secret-token",
	}, nil, nil, []checkResult{{Name: "context_config", Status: statusPass}})

	data, err := json.Marshal(output)

	require.NoError(t, err)
	assert.NotContains(t, string(data), "secret-token")
	assert.Contains(t, string(data), "directory.example.com:443")
}

func TestDetailKeysPrioritizesErrorsAndSortsRest(t *testing.T) {
	keys := detailKeys(map[string]string{
		"zeta":        "z",
		"error":       "boom",
		"alpha":       "a",
		"close_error": "close",
	})

	assert.Equal(t, []string{"error", "close_error", "alpha", "zeta"}, keys)
	assert.Nil(t, detailKeys(nil))
}

func TestPrintOutputFormats(t *testing.T) {
	output := doctorOutput{
		Config: configInfo{Context: "dev", ContextSource: "current_context", ServerAddress: "localhost:8888", BootstrapPeers: 1},
		Results: []checkResult{
			{Name: "context_config", Status: statusPass, Message: "ok"},
			{Name: "directory_api_tcp", Status: statusFail, Message: "failed", Details: map[string]string{"error": "boom"}},
		},
	}
	output.Summary = summarize(output.Results)

	tests := []struct {
		name     string
		format   presenter.OutputFormat
		contains string
	}{
		{name: "human", format: presenter.FormatHuman, contains: "dirctl doctor"},
		{name: "json", format: presenter.FormatJSON, contains: "\"summary\""},
		{name: "jsonl", format: presenter.FormatJSONL, contains: "\"summary\""},
		{name: "raw", format: presenter.FormatRaw, contains: "1 passed, 1 failed, 0 warned, 0 skipped"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, out := newOutputTestCommand(t, tt.format)

			require.NoError(t, printOutput(cmd, output))
			assert.Contains(t, out.String(), tt.contains)
		})
	}
}

func TestPrintOutputRejectsUnsupportedFormat(t *testing.T) {
	cmd, _ := newOutputTestCommand(t, "xml")

	err := printOutput(cmd, doctorOutput{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported output format")
}

func newOutputTestCommand(t *testing.T, format presenter.OutputFormat) (*cobra.Command, *bytes.Buffer) {
	t.Helper()

	cmd := &cobra.Command{}
	out := &bytes.Buffer{}
	cmd.SetOut(out)
	presenter.AddOutputFlags(cmd)
	require.NoError(t, cmd.Flags().Set("output", string(format)))

	return cmd, out
}
