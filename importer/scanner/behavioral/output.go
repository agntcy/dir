// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package behavioral

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/agntcy/dir/importer/scanner/types"
)

// mcpScannerResult represents a single tool result from `mcp-scanner behavioral --raw`.
type mcpScannerResult struct {
	ToolName string                       `json:"tool_name"`
	Status   string                       `json:"status"`
	IsSafe   bool                         `json:"is_safe"`
	Findings map[string]mcpAnalyzerResult `json:"findings"`
}

// mcpAnalyzerResult represents the output of a single analyzer within mcp-scanner.
type mcpAnalyzerResult struct {
	Severity      string   `json:"severity"`
	ThreatSummary string   `json:"threat_summary"`
	ThreatNames   []string `json:"threat_names"`
	TotalFindings int      `json:"total_findings"`
}

// parseOutput parses the JSON output of `mcp-scanner behavioral --raw`
// and maps it to a ScanResult.
func parseOutput(raw []byte) (*types.ScanResult, error) {
	raw = trimToJSON(raw)

	var results []mcpScannerResult
	if err := json.Unmarshal(raw, &results); err != nil {
		return nil, fmt.Errorf("parse mcp-scanner output: %w", err)
	}

	if len(results) == 0 {
		return &types.ScanResult{Safe: true}, nil
	}

	var findings []types.Finding

	for _, r := range results {
		if r.IsSafe {
			continue
		}

		for analyzerName, ar := range r.Findings {
			severity := mapSeverity(ar.Severity)
			msg := fmt.Sprintf("[%s] %s: %s",
				analyzerName,
				r.ToolName,
				ar.ThreatSummary,
			)

			if len(ar.ThreatNames) > 0 {
				msg += " (" + strings.Join(ar.ThreatNames, ", ") + ")"
			}

			findings = append(findings, types.Finding{
				Severity: severity,
				Message:  msg,
			})
		}
	}

	return &types.ScanResult{
		Safe:     len(findings) == 0,
		Findings: findings,
	}, nil
}

// trimToJSON strips any leading non-JSON content (e.g. log lines on stderr
// that leaked into stdout) by finding the first '['.
func trimToJSON(raw []byte) []byte {
	idx := bytes.IndexByte(raw, '[')
	if idx > 0 {
		return raw[idx:]
	}

	return raw
}

func mapSeverity(s string) types.FindingSeverity {
	switch strings.ToUpper(s) {
	case "CRITICAL", "HIGH":
		return types.SeverityError
	case "MEDIUM":
		return types.SeverityWarning
	default:
		return types.SeverityInfo
	}
}
