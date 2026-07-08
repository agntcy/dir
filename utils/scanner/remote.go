// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package scanner

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/utils/logging"
	"google.golang.org/protobuf/types/known/structpb"
)

// remoteSubcommands are the mcp-scanner live-server subcommands run against
// every discovered remote MCP endpoint.
var remoteSubcommands = []string{"remote", "prompts", "resources", "instructions"}

var remoteLogger = logging.Logger("utils/scanner/remote")

// RemoteConfig holds configuration for the Remote runner.
type RemoteConfig struct {
	// CLIPath is the path to the mcp-scanner binary. Defaults to DefaultMCPCLIPath.
	CLIPath string
}

// RemoteRunner invokes mcp-scanner against live MCP server endpoints declared
// on the record, running the remote, prompts, resources, and instructions
// subcommands against each one and merging the findings into a single
// ScanResult.
//
// Unlike MCPRunner (which clones and scans source code), RemoteRunner never
// fails hard on network errors: an unreachable endpoint or auth failure for a
// given (endpoint, subcommand) pair is recorded as a skipped sub-scan with a
// warning logged, so one bad endpoint never blocks the others.
type RemoteRunner struct {
	cfg RemoteConfig
}

// NewRemoteRunner creates a RemoteRunner. If cfg.CLIPath is empty, DefaultMCPCLIPath is used.
func NewRemoteRunner(cfg RemoteConfig) *RemoteRunner {
	if cfg.CLIPath == "" {
		cfg.CLIPath = DefaultMCPCLIPath
	}

	return &RemoteRunner{cfg: cfg}
}

// Name returns the runner name.
func (r *RemoteRunner) Name() string { return "remote" }

// Run extracts remote MCP endpoint URLs from the record and runs the remote,
// prompts, resources, and instructions subcommands against each one, merging
// the findings into a single ScanResult. If no remote-capable endpoint is
// found on the record, the scan is skipped (this is the normal case for
// records with a stdio-only or source-code-only MCP server).
func (r *RemoteRunner) Run(ctx context.Context, record *corev1.Record) (*ScanResult, error) {
	endpoints := extractRemoteEndpoints(record)
	if len(endpoints) == 0 {
		return &ScanResult{Skipped: true, SkippedReason: "no remote MCP endpoint found"}, nil
	}

	results := make([]*ScanResult, 0, len(endpoints)*len(remoteSubcommands))

	for _, url := range endpoints {
		for _, sub := range remoteSubcommands {
			results = append(results, r.runOne(ctx, sub, url))
		}
	}

	return merge(results), nil
}

// runOne invokes a single mcp-scanner live-server subcommand against one
// endpoint. Any failure — unreachable server, auth failure, or unparsable
// output — is returned as a skipped ScanResult with a warning logged, rather
// than surfaced as an error, per the issue's "skipped with a warning, not a
// hard error" requirement for network failures.
func (r *RemoteRunner) runOne(ctx context.Context, subcommand, url string) *ScanResult {
	rawOutput, err := runMCPScannerRemote(ctx, r.cfg.CLIPath, subcommand, url)
	if err != nil {
		remoteLogger.Warn("mcp-scanner remote subcommand failed, skipping", "subcommand", subcommand, "url", url, "error", err)

		return &ScanResult{
			Skipped:       true,
			SkippedReason: fmt.Sprintf("%s %s: %s", subcommand, url, err),
		}
	}

	result, err := parseMCPOutput(rawOutput)
	if err != nil {
		remoteLogger.Warn("mcp-scanner remote subcommand produced unparsable output, skipping", "subcommand", subcommand, "url", url, "error", err)

		return &ScanResult{
			Skipped:       true,
			SkippedReason: fmt.Sprintf("%s %s: unparsable output: %s", subcommand, url, err),
		}
	}

	result.Findings = tagFindings(subcommand, url, result.Findings)
	result.Analyzers = []string{subcommand}

	return result
}

// tagFindings prefixes each finding's message with the subcommand and
// endpoint URL it came from, so findings merged from multiple endpoints and
// subcommands remain individually traceable.
func tagFindings(subcommand, url string, findings []Finding) []Finding {
	if len(findings) == 0 {
		return findings
	}

	tagged := make([]Finding, len(findings))
	for i, f := range findings {
		tagged[i] = Finding{
			Severity: f.Severity,
			Message:  fmt.Sprintf("[%s %s] %s", subcommand, url, f.Message),
		}
	}

	return tagged
}

func runMCPScannerRemote(ctx context.Context, cliPath, subcommand, serverURL string) ([]byte, error) {
	var stdout, stderr bytes.Buffer

	// mcp-scanner requires the global --raw flag to precede the subcommand;
	// only subcommand-specific flags (--server-url) follow it.
	cmd := exec.CommandContext(ctx, cliPath, "--raw", subcommand, "--server-url", serverURL)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("mcp-scanner %s exited with error: %s: %w", subcommand, strings.TrimSpace(stderr.String()), err)
	}

	return stdout.Bytes(), nil
}

// extractRemoteEndpoints decodes the record and returns the URLs of every
// remote-capable MCP connection (transport type "sse" or "streamable-http")
// declared across all of the record's modules.
func extractRemoteEndpoints(record *corev1.Record) []string {
	if record == nil {
		return nil
	}

	decoded, err := record.Decode()
	if err != nil {
		return nil
	}

	if !decoded.HasV1() {
		return nil
	}

	var urls []string

	for _, mod := range decoded.GetV1().GetModules() {
		urls = append(urls, extractConnectionURLs(mod.GetData())...)
	}

	return urls
}

// extractConnectionURLs walks data.mcp_data.connections[] and returns the
// url of every connection whose transport type is remote-capable ("sse" or
// "streamable-http"). "stdio" connections are local (spawned via command)
// and have no endpoint to scan.
func extractConnectionURLs(data *structpb.Struct) []string {
	conns := data.GetFields()["mcp_data"].GetStructValue().GetFields()["connections"].GetListValue().GetValues()

	var urls []string

	for _, c := range conns {
		conn := c.GetStructValue()
		if conn == nil {
			continue
		}

		switch conn.GetFields()["type"].GetStringValue() {
		case "sse", "streamable-http":
		default:
			continue
		}

		if url := conn.GetFields()["url"].GetStringValue(); url != "" {
			urls = append(urls, url)
		}
	}

	return urls
}
