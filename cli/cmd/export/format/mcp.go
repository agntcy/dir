// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package format

import (
	"encoding/json"
	"fmt"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/oasf-sdk/pkg/translator"
)

func init() {
	RegisterFormatter("mcp-ghcopilot", &mcpGHCopilotFormatter{})
}

type mcpGHCopilotFormatter struct{}

func (f *mcpGHCopilotFormatter) Format(record *corev1.Record) ([]byte, error) {
	data := record.GetData()
	if data == nil {
		return nil, fmt.Errorf("record contains no data")
	}

	ghCopilotConfig, err := translator.RecordToGHCopilot(data)
	if err != nil {
		return nil, fmt.Errorf("failed to translate record to GitHub Copilot MCP config: %w", err)
	}

	raw, err := json.MarshalIndent(ghCopilotConfig, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal GitHub Copilot MCP config to JSON: %w", err)
	}

	raw = append(raw, '\n')

	return raw, nil
}

func (f *mcpGHCopilotFormatter) FileExtension() string {
	return ExtJSON
}
