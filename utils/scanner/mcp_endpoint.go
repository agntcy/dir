// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package scanner

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"net/url"
	"os/exec"
	"strings"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"google.golang.org/protobuf/types/known/structpb"
)

// endpointSubcommands are the mcp-scanner live-server subcommands run against
// every discovered endpoint.
var endpointSubcommands = []string{"remote", "prompts", "resources", "instructions"}

// runEndpointScan scans every live MCP endpoint the record declares, merging
// the findings into one result. Endpoint URLs come from publisher-controlled
// record data, so each is validated before it reaches mcp-scanner.
//
// This phase never returns an error: an unreachable endpoint, an auth failure,
// or a rejected URL is recorded as a skipped sub-scan with a warning logged,
// so one bad endpoint never blocks the others or the source scan.
func (r *MCPRunner) runEndpointScan(ctx context.Context, record *corev1.Record) *ScanResult {
	endpoints := extractEndpoints(record)
	if len(endpoints) == 0 {
		return &ScanResult{Skipped: true, SkippedReason: "no remote MCP endpoint found"}
	}

	results := make([]*ScanResult, 0, len(endpoints)*len(endpointSubcommands))

	for _, endpoint := range endpoints {
		if err := validateEndpointURL(endpoint, r.cfg.AllowPrivateEndpoints); err != nil {
			mcpLogger.Warn("rejected MCP endpoint URL, skipping", "url", endpoint, "error", err)

			results = append(results, &ScanResult{
				Skipped:       true,
				SkippedReason: fmt.Sprintf("%s: %s", endpoint, err),
			})

			continue
		}

		for _, sub := range endpointSubcommands {
			results = append(results, r.runEndpointSubcommand(ctx, sub, endpoint))
		}
	}

	return merge(results)
}

// runEndpointSubcommand invokes a single mcp-scanner live-server subcommand
// against one endpoint. Any failure is returned as a skipped ScanResult with a
// warning logged rather than surfaced as an error.
func (r *MCPRunner) runEndpointSubcommand(ctx context.Context, subcommand, endpoint string) *ScanResult {
	rawOutput, err := runMCPScannerEndpoint(ctx, r.cfg.CLIPath, subcommand, endpoint)
	if err != nil {
		mcpLogger.Warn("mcp-scanner endpoint subcommand failed, skipping", "subcommand", subcommand, "url", endpoint, "error", err)

		return &ScanResult{
			Skipped:       true,
			SkippedReason: fmt.Sprintf("%s %s: %s", subcommand, endpoint, err),
		}
	}

	result, err := parseMCPOutput(rawOutput)
	if err != nil {
		mcpLogger.Warn("mcp-scanner endpoint subcommand produced unparsable output, skipping", "subcommand", subcommand, "url", endpoint, "error", err)

		return &ScanResult{
			Skipped:       true,
			SkippedReason: fmt.Sprintf("%s %s: unparsable output: %s", subcommand, endpoint, err),
		}
	}

	result.Findings = tagFindings(subcommand, endpoint, result.Findings)
	result.Analyzers = []string{subcommand}

	return result
}

// tagFindings prefixes each finding's message with the subcommand and endpoint
// it came from, so findings merged across endpoints and subcommands remain
// individually traceable.
func tagFindings(subcommand, endpoint string, findings []Finding) []Finding {
	if len(findings) == 0 {
		return findings
	}

	tagged := make([]Finding, len(findings))
	for i, f := range findings {
		tagged[i] = Finding{
			Severity: f.Severity,
			Message:  fmt.Sprintf("[%s %s] %s", subcommand, endpoint, f.Message),
		}
	}

	return tagged
}

func runMCPScannerEndpoint(ctx context.Context, cliPath, subcommand, serverURL string) ([]byte, error) {
	var stdout, stderr bytes.Buffer

	// mcp-scanner requires the global --raw flag to precede the subcommand;
	// only subcommand-specific flags (--server-url) follow it.
	cmd := exec.CommandContext(ctx, cliPath, "--raw", subcommand, "--server-url", serverURL) //nolint:gosec
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("mcp-scanner %s exited with error: %s: %w", subcommand, strings.TrimSpace(stderr.String()), err)
	}

	return stdout.Bytes(), nil
}

// validateEndpointURL rejects endpoint URLs that should never be dialled from
// the reconciler. Endpoint URLs arrive in externally published records, so an
// unchecked value points the scanner wherever the publisher likes, including
// cloud metadata services and hosts reachable only from inside the cluster.
//
// This is a mitigation and not a complete defence. dir validates the URL, but
// the connection is opened by the mcp-scanner process, so a hostname that
// resolves to a private address only at dial time still gets through. Closing
// that gap needs control over the dialer, which lives in mcp-scanner.
func validateEndpointURL(raw string, allowPrivate bool) error {
	parsed, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("unparsable URL: %w", err)
	}

	// https only by default. Plain http is allowed alongside private ranges,
	// because the deployments that need one generally need the other.
	switch parsed.Scheme {
	case "https":
	case "http":
		if !allowPrivate {
			return fmt.Errorf("scheme %q not allowed (https only)", parsed.Scheme)
		}
	default:
		return fmt.Errorf("scheme %q not allowed (https only)", parsed.Scheme)
	}

	host := parsed.Hostname()
	if host == "" {
		return fmt.Errorf("URL has no host")
	}

	if allowPrivate {
		return nil
	}

	// A literal IP can be checked directly. A hostname is left to resolve at
	// dial time; see the doc comment on the residual risk.
	if ip := net.ParseIP(host); ip != nil && !isPublicIP(ip) {
		return fmt.Errorf("endpoint address %s is not publicly routable", ip)
	}

	if isLoopbackHostname(host) {
		return fmt.Errorf("endpoint host %q is loopback", host)
	}

	return nil
}

// isPublicIP reports whether ip is routable on the public internet. It rejects
// loopback, link-local (which covers the 169.254.169.254 cloud metadata
// address), private/unique-local, unspecified, and multicast addresses.
func isPublicIP(ip net.IP) bool {
	switch {
	case ip.IsLoopback(),
		ip.IsLinkLocalUnicast(),
		ip.IsLinkLocalMulticast(),
		ip.IsInterfaceLocalMulticast(),
		ip.IsMulticast(),
		ip.IsUnspecified(),
		ip.IsPrivate():
		return false
	}

	return true
}

// isLoopbackHostname catches the names that resolve to loopback on every
// machine without a lookup. Other hostnames are not resolved here: a DNS
// lookup at validation time can disagree with the one at dial time, so it
// would buy confidence the check cannot honour.
func isLoopbackHostname(host string) bool {
	switch strings.ToLower(strings.TrimSuffix(host, ".")) {
	case "localhost", "localhost.localdomain", "ip6-localhost", "ip6-loopback":
		return true
	}

	return strings.HasSuffix(strings.ToLower(host), ".localhost")
}

// extractEndpoints decodes the record and returns the URLs of every
// remote-capable MCP connection declared across all of the record's modules.
func extractEndpoints(record *corev1.Record) []string {
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

// extractConnectionURLs walks data.connections[] and returns the url of every
// connection whose transport type is remote-capable ("sse" or
// "streamable-http"). A module's data is itself the OASF mcp_data object, so
// connections live directly under it. "stdio" connections are local (spawned
// via command) and have no endpoint to scan.
func extractConnectionURLs(data *structpb.Struct) []string {
	conns := data.GetFields()["connections"].GetListValue().GetValues()

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
