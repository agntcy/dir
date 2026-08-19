// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package scanner

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/netip"
	"net/url"
	"os"
	"os/exec"
	"strings"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"google.golang.org/protobuf/types/known/structpb"
)

// endpointSubcommands are the mcp-scanner live-server subcommands run against
// every discovered endpoint.
var endpointSubcommands = []string{"remote", "prompts", "resources", "instructions"}

// DefaultMaxEndpointsPerRecord bounds how many endpoints one record can make
// the reconciler dial.
//
// Validating the destination is not enough on its own: the endpoint list is
// publisher-controlled and each entry costs len(endpointSubcommands)
// invocations, so an unbounded loop lets one record turn the directory into an
// amplifier. A record declaring 500 connections would fire 2000 outbound scans
// at hosts of the publisher's choosing. Eight is well above what a real MCP
// server declares (stdio plus one or two remote transports) and far below a
// useful amplification factor.
const DefaultMaxEndpointsPerRecord = 8

// runEndpointScan scans the live MCP endpoints the record declares, merging
// the findings into one result. Endpoint URLs come from publisher-controlled
// record data, so the list is capped and each URL is validated before it
// reaches mcp-scanner.
//
// This phase never returns an error: an unreachable endpoint, an auth failure,
// or a rejected URL is recorded as a skipped sub-scan with a warning logged,
// so one bad endpoint never blocks the others or the source scan.
func (r *MCPRunner) runEndpointScan(ctx context.Context, record *corev1.Record) *ScanResult {
	endpoints := extractEndpoints(record)
	if len(endpoints) == 0 {
		return &ScanResult{Skipped: true, SkippedReason: "no remote MCP endpoint found"}
	}

	maxEndpoints := r.cfg.MaxEndpointsPerRecord
	if maxEndpoints <= 0 {
		maxEndpoints = DefaultMaxEndpointsPerRecord
	}

	var notices []string

	if len(endpoints) > maxEndpoints {
		mcpLogger.Warn("record declares more MCP endpoints than the per-record cap, truncating",
			"declared", len(endpoints), "cap", maxEndpoints)

		notices = append(notices, fmt.Sprintf("record declares %d endpoints, scanned the first %d",
			len(endpoints), maxEndpoints))

		endpoints = endpoints[:maxEndpoints]
	}

	results := make([]*ScanResult, 0, len(endpoints)*len(endpointSubcommands))

	for _, endpoint := range endpoints {
		if err := validateEndpointURL(endpoint, r.cfg.AllowPrivateEndpoints, r.cfg.AllowInsecureTransport); err != nil {
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

	result := merge(results)

	// Notices, not SkippedReason. merge propagates SkippedReason only when
	// every input skipped, so a truncation note written there vanishes as soon
	// as one endpoint scans successfully - and again when Run merges this
	// result with the source phase. Notices survive both merges. A Finding
	// would be wrong for the opposite reason: any finding sets Safe=false, and
	// reduced coverage is not a security defect.
	result.Notices = append(result.Notices, notices...)

	return result
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

// endpointAnalyzers returns the analyzer set for the live-endpoint phase.
// mcp-scanner aborts this path when a requested analyzer's credentials are
// missing, so its "api,yara,llm" default fails every scan on a deployment
// without a Cisco AI Defense key. yara and readiness need no credentials and
// always run; llm and api stay opt-in until their key is configured.
func endpointAnalyzers() []string {
	analyzers := []string{"yara", "readiness"}

	if llmConfigured() {
		analyzers = append(analyzers, "llm")
	}

	if os.Getenv("MCP_SCANNER_API_KEY") != "" {
		analyzers = append(analyzers, "api")
	}

	return analyzers
}

func runMCPScannerEndpoint(ctx context.Context, cliPath, subcommand, serverURL string) ([]byte, error) {
	var stdout, stderr bytes.Buffer

	// mcp-scanner requires global flags (--analyzers, --raw) to precede the
	// subcommand; only subcommand-specific flags (--server-url) follow it.
	cmd := exec.CommandContext(ctx, cliPath, //nolint:gosec
		"--analyzers", strings.Join(endpointAnalyzers(), ","),
		"--raw", subcommand,
		"--server-url", serverURL,
	)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	// Maps AZURE_OPENAI_* onto the MCP_SCANNER_LLM_* names the scanner reads.
	cmd.Env = buildMCPScannerEnv()

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("mcp-scanner %s exited with error: %s: %w", subcommand, strings.TrimSpace(stderr.String()), err)
	}

	return stdout.Bytes(), nil
}

// reservedPrefixes are the address ranges an endpoint must never resolve to.
//
// This is deliberately an explicit table rather than the net.IP predicates.
// net.IP.IsPrivate covers only RFC 1918 and fc00::/7, which leaves live
// metadata and internal-network ranges accepted: 100.100.100.200 is Alibaba
// Cloud's instance metadata service, and 100.64.0.0/10 is the pod and node
// CIDR on several managed Kubernetes offerings, which is exactly the
// "reachable only from inside the cluster" case this check exists to stop.
var reservedPrefixes = []netip.Prefix{
	// IPv4.
	netip.MustParsePrefix("0.0.0.0/8"),       // "this network"
	netip.MustParsePrefix("10.0.0.0/8"),      // RFC 1918
	netip.MustParsePrefix("100.64.0.0/10"),   // RFC 6598 CGNAT, incl. Alibaba metadata
	netip.MustParsePrefix("127.0.0.0/8"),     // loopback
	netip.MustParsePrefix("169.254.0.0/16"),  // link-local, incl. 169.254.169.254
	netip.MustParsePrefix("172.16.0.0/12"),   // RFC 1918
	netip.MustParsePrefix("192.0.0.0/24"),    // IETF protocol assignments
	netip.MustParsePrefix("192.0.2.0/24"),    // TEST-NET-1
	netip.MustParsePrefix("192.88.99.0/24"),  // 6to4 relay anycast
	netip.MustParsePrefix("192.168.0.0/16"),  // RFC 1918
	netip.MustParsePrefix("198.18.0.0/15"),   // RFC 2544 benchmarking
	netip.MustParsePrefix("198.51.100.0/24"), // TEST-NET-2
	netip.MustParsePrefix("203.0.113.0/24"),  // TEST-NET-3
	netip.MustParsePrefix("224.0.0.0/4"),     // multicast
	netip.MustParsePrefix("240.0.0.0/4"),     // reserved, incl. 255.255.255.255
	// IPv6.
	netip.MustParsePrefix("::/96"),         // unspecified + deprecated IPv4-compatible (::127.0.0.1)
	netip.MustParsePrefix("::1/128"),       // loopback
	netip.MustParsePrefix("64:ff9b::/96"),  // NAT64, can embed a v4 target
	netip.MustParsePrefix("100::/64"),      // discard-only
	netip.MustParsePrefix("2001::/32"),     // Teredo
	netip.MustParsePrefix("2001:db8::/32"), // documentation
	netip.MustParsePrefix("2002::/16"),     // 6to4, can embed a v4 target
	netip.MustParsePrefix("fc00::/7"),      // unique local
	netip.MustParsePrefix("fe80::/10"),     // link-local
	netip.MustParsePrefix("ff00::/8"),      // multicast
}

// loopbackHostnames resolve to loopback on essentially every machine, so they
// are refused without a lookup.
var loopbackHostnames = map[string]struct{}{
	"localhost":             {},
	"localhost.localdomain": {},
	"ip6-localhost":         {},
	"ip6-loopback":          {},
}

// validateEndpointURL rejects endpoint URLs that should never be dialled from
// the reconciler. Endpoint URLs arrive in externally published records, so an
// unchecked value points the scanner wherever the publisher likes, including
// cloud metadata services and hosts reachable only from inside the cluster.
//
// The host must be either a canonical IP literal outside reservedPrefixes or a
// syntactically valid DNS name. Anything else is refused. That second rule is
// load-bearing: the resolver accepts numeric host forms netip does not parse
// (decimal 2852039166, octal 0177.0.0.1, hex 0x7f000001, short 127.1), and
// treating a failed IP parse as "then it must be a hostname" waves every one
// of them through to a loopback or metadata address.
//
// This remains a mitigation, not a complete defence. dir validates the URL,
// but the connection is opened by the mcp-scanner process, so a DNS name that
// resolves to a reserved address at dial time, or a redirect chain ending at
// one, still gets through. Closing that needs control over the dialer, which
// lives in mcp-scanner.
func validateEndpointURL(raw string, allowPrivate, allowInsecure bool) error {
	parsed, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("unparsable URL: %w", err)
	}

	switch parsed.Scheme {
	case "https":
	case "http":
		if !allowInsecure {
			return fmt.Errorf("scheme %q not allowed; enable allow_insecure_transport to permit it", parsed.Scheme)
		}
	default:
		return fmt.Errorf("scheme %q not allowed (http and https only)", parsed.Scheme)
	}

	// Normalize the fully-qualified trailing dot once, up front. Applying it
	// to only one of the two paths below is how "169.254.169.254." ends up
	// taking a different branch from "169.254.169.254".
	host := strings.ToLower(strings.TrimSuffix(parsed.Hostname(), "."))
	if host == "" {
		return errors.New("URL has no host")
	}

	if addr, parseErr := netip.ParseAddr(host); parseErr == nil {
		// A zone (fe80::1%eth0) is meaningful only on the dialling host and is
		// not representable in the prefix table, so refuse it rather than let
		// it skip the range check.
		if addr.Zone() != "" {
			return fmt.Errorf("endpoint address %q carries an interface zone", host)
		}

		// ::ffff:127.0.0.1 has to be range-checked as 127.0.0.1.
		addr = addr.Unmap()

		if !allowPrivate && isReservedAddr(addr) {
			return fmt.Errorf("endpoint address %s is not publicly routable", addr)
		}

		return nil
	}

	if !isValidDNSName(host) {
		return fmt.Errorf("endpoint host %q is neither a valid IP address nor a valid DNS name", host)
	}

	if !allowPrivate {
		if _, ok := loopbackHostnames[host]; ok {
			return fmt.Errorf("endpoint host %q is loopback", host)
		}

		if strings.HasSuffix(host, ".localhost") {
			return fmt.Errorf("endpoint host %q is loopback", host)
		}
	}

	return nil
}

// isReservedAddr reports whether addr falls in a range an endpoint must not
// resolve to.
func isReservedAddr(addr netip.Addr) bool {
	for _, p := range reservedPrefixes {
		// Prefix.Contains is family-strict, so a v4 address never matches a v6
		// prefix and no explicit family guard is needed.
		if p.Contains(addr) {
			return true
		}
	}

	return false
}

// isValidDNSName reports whether host is a syntactically valid DNS name.
//
// The all-numeric rule is the load-bearing one: a label set that is entirely
// digits is not a hostname, it is an integer the resolver reads as an IPv4
// address (2852039166 is 169.254.169.254, 127.1 is 127.0.0.1). The 0x-prefixed
// check covers the hex spelling of the same trick.
func isValidDNSName(host string) bool {
	const maxNameLen = 253

	if len(host) > maxNameLen {
		return false
	}

	allNumeric := true

	for label := range strings.SplitSeq(host, ".") {
		numeric, ok := classifyDNSLabel(label)
		if !ok {
			return false
		}

		if !numeric {
			allNumeric = false
		}
	}

	return !allNumeric
}

// classifyDNSLabel reports whether label is a legal DNS label, and whether it
// is entirely digits.
func classifyDNSLabel(label string) (numeric, ok bool) { //nolint:nonamedreturns // the two bools are indistinguishable without names
	const maxLabelLen = 63

	if label == "" || len(label) > maxLabelLen {
		return false, false
	}

	if label[0] == '-' || label[len(label)-1] == '-' {
		return false, false
	}

	// The hex spelling of an integer address (0x7f000001) is a legal-looking
	// label that the resolver reads as an IPv4 address.
	if strings.HasPrefix(label, "0x") && isHexDigits(label[2:]) {
		return false, false
	}

	numeric = true

	for i := range len(label) {
		c := label[i]

		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c == '-':
			numeric = false
		case c >= '0' && c <= '9':
		default:
			return false, false
		}
	}

	return numeric, true
}

// isHexDigits reports whether s is a non-empty run of hex digits.
func isHexDigits(s string) bool {
	if s == "" {
		return false
	}

	for i := range len(s) {
		c := s[i]
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') && (c < 'A' || c > 'F') {
			return false
		}
	}

	return true
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

	seen := make(map[string]struct{})

	for _, mod := range decoded.GetV1().GetModules() {
		for _, u := range extractConnectionURLs(mod.GetData()) {
			// Deduplicate before the cap sees the list. Without this, a record
			// declaring the same URL eight times stays under the cap and still
			// produces four requests per copy at a single host, so the cap
			// would bound list length without bounding what any one victim
			// receives.
			if _, dup := seen[u]; dup {
				continue
			}

			seen[u] = struct{}{}

			urls = append(urls, u)
		}
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

		// Named u, not url: this file imports net/url.
		if u := conn.GetFields()["url"].GetStringValue(); u != "" {
			urls = append(urls, u)
		}
	}

	return urls
}
