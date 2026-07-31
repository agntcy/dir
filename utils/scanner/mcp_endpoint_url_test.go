// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package scanner

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

// Endpoint URLs arrive in externally published records, so validateEndpointURL
// is the boundary between publisher-controlled data and an outbound connection
// made by the reconciler. These cases pin the rejections that matter.
func TestValidateEndpointURL_Rejected(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		url  string
	}{
		{"cloud metadata address", "https://169.254.169.254/latest/meta-data/"},
		{"metadata over http", "http://169.254.169.254/"},
		{"loopback v4", "https://127.0.0.1:8080/mcp"},
		{"loopback v4 alternate", "https://127.1.2.3/mcp"},
		{"loopback v6", "https://[::1]/mcp"},
		{"localhost", "https://localhost:3000/mcp"},
		{"localhost trailing dot", "https://localhost./mcp"},
		{"localhost uppercase", "https://LOCALHOST/mcp"},
		{"localhost subdomain", "https://foo.localhost/mcp"},
		{"private 10/8", "https://10.0.0.5/mcp"},
		{"private 172.16/12", "https://172.16.4.9/mcp"},
		{"private 192.168/16", "https://192.168.1.1/mcp"},
		{"unique local v6", "https://[fd00::1]/mcp"},
		{"link-local v6", "https://[fe80::1]/mcp"},
		{"unspecified", "https://0.0.0.0/mcp"},
		{"multicast", "https://224.0.0.1/mcp"},
		{"plain http", "http://mcp.example.com/mcp"},
		{"file scheme", "file:///etc/passwd"},
		{"gopher scheme", "gopher://example.com/"},
		{"no scheme", "example.com/mcp"},
		{"no host", "https:///mcp"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if err := validateEndpointURL(tc.url, false, false); err == nil {
				t.Errorf("validateEndpointURL(%q) = nil, want rejection", tc.url)
			}
		})
	}
}

func TestValidateEndpointURL_Accepted(t *testing.T) {
	t.Parallel()

	cases := []string{
		"https://mcp.example.com/mcp",
		"https://mcp.example.com:8443/sse",
		"https://8.8.8.8/mcp",
		"https://[2606:4700:4700::1111]/mcp",
	}

	for _, u := range cases {
		t.Run(u, func(t *testing.T) {
			t.Parallel()

			if err := validateEndpointURL(u, false, false); err != nil {
				t.Errorf("validateEndpointURL(%q) = %v, want accepted", u, err)
			}
		})
	}
}

// AllowPrivateEndpoints is the on-prem escape hatch: the directory and its MCP
// servers may legitimately share a private network, where the default posture
// would reject every endpoint.
func TestValidateEndpointURL_AllowPrivate(t *testing.T) {
	t.Parallel()

	cases := []string{
		"https://10.0.0.5/mcp",
		"https://localhost:3000/mcp",
		"http://192.168.1.1/mcp",
		"http://mcp.internal/mcp",
	}

	for _, u := range cases {
		t.Run(u, func(t *testing.T) {
			t.Parallel()

			if err := validateEndpointURL(u, true, true); err != nil {
				t.Errorf("validateEndpointURL(%q, allowPrivate) = %v, want accepted", u, err)
			}
		})
	}
}

// Even with private ranges allowed, a scheme that is not http(s) stays out:
// the allowance is about reachability, not about handing arbitrary schemes to
// the scanner.
func TestValidateEndpointURL_AllowPrivate_StillRejectsNonHTTPSchemes(t *testing.T) {
	t.Parallel()

	for _, u := range []string{"file:///etc/passwd", "gopher://10.0.0.1/"} {
		if err := validateEndpointURL(u, true, true); err == nil {
			t.Errorf("validateEndpointURL(%q, allowPrivate) = nil, want rejection", u)
		}
	}
}

// A rejected endpoint must be skipped with a reason rather than silently
// dropped or escalated to an error, and it must not stop the other endpoints
// on the same record from being scanned.
func TestRunEndpointScan_RejectedURL_SkippedAndOthersStillScanned(t *testing.T) {
	t.Parallel()

	r := NewMCPRunner(MCPConfig{CLIPath: fakeCLIPath(t)})
	rec := recordWithEndpoints(t, "https://169.254.169.254/latest/", "https://ok.example.com/mcp")

	got := r.runEndpointScan(context.Background(), rec)

	if got.Skipped {
		t.Fatalf("the reachable endpoint should still produce a result, got skipped: %s", got.SkippedReason)
	}

	// 1 endpoint x 4 subcommands, each producing one finding from the fake CLI.
	// The rejected endpoint contributes a skip, not findings.
	const wantFindings = 4

	if len(got.Findings) != wantFindings {
		t.Errorf("want %d findings from the surviving endpoint, got %d: %+v", wantFindings, len(got.Findings), got.Findings)
	}

	for _, f := range got.Findings {
		if strings.Contains(f.Message, "169.254.169.254") {
			t.Errorf("rejected endpoint must not be scanned, but a finding references it: %q", f.Message)
		}
	}
}

func TestRunEndpointScan_AllURLsRejected_SkippedNotError(t *testing.T) {
	t.Parallel()

	r := NewMCPRunner(MCPConfig{CLIPath: fakeCLIPath(t)})
	rec := recordWithEndpoints(t, "https://127.0.0.1/mcp", "file:///etc/passwd")

	got := r.runEndpointScan(context.Background(), rec)

	if !got.Skipped {
		t.Fatalf("every endpoint was rejected, so the phase should skip: %+v", got)
	}

	if got.SkippedReason == "" {
		t.Error("a skip caused by rejected URLs should carry a reason")
	}
}

// Validating the destination bounds WHERE the reconciler connects but not HOW
// MANY connections one record can cause. The endpoint list is
// publisher-controlled and each entry costs four invocations, so without a cap
// a single record turns the directory into an amplifier.
func TestRunEndpointScan_CapsEndpointsPerRecord(t *testing.T) {
	t.Parallel()

	const declared = 20

	urls := make([]string, 0, declared)
	for i := range declared {
		urls = append(urls, fmt.Sprintf("https://mcp%d.example.com/mcp", i))
	}

	r := NewMCPRunner(MCPConfig{CLIPath: fakeCLIPath(t), MaxEndpointsPerRecord: 3})
	got := r.runEndpointScan(context.Background(), recordWithEndpoints(t, urls...))

	// 3 endpoints x 4 subcommands, one finding each from the fake CLI.
	const wantFindings = 3 * 4

	if len(got.Findings) != wantFindings {
		t.Errorf("cap not applied: want %d findings, got %d", wantFindings, len(got.Findings))
	}

	// Truncation has to be visible, or a report covering 3 of 20 endpoints
	// reads the same as one covering everything the record declared.
	if len(got.Notices) != 1 || !strings.Contains(got.Notices[0], "20") || !strings.Contains(got.Notices[0], "3") {
		t.Errorf("truncation should be recorded with both counts, got %v", got.Notices)
	}
}

func TestRunEndpointScan_DefaultCapApplies(t *testing.T) {
	t.Parallel()

	urls := make([]string, 0, DefaultMaxEndpointsPerRecord+5)
	for i := range DefaultMaxEndpointsPerRecord + 5 {
		urls = append(urls, fmt.Sprintf("https://mcp%d.example.com/mcp", i))
	}

	// No MaxEndpointsPerRecord set: the zero value must fall back to the
	// default rather than meaning "no limit".
	r := NewMCPRunner(MCPConfig{CLIPath: fakeCLIPath(t)})
	got := r.runEndpointScan(context.Background(), recordWithEndpoints(t, urls...))

	wantFindings := DefaultMaxEndpointsPerRecord * len(endpointSubcommands)
	if len(got.Findings) != wantFindings {
		t.Errorf("zero value should apply the default cap: want %d findings, got %d", wantFindings, len(got.Findings))
	}
}

// Exactly at the cap is the boundary that matters: an off-by-one turns a
// fully-scanned record into one that reports itself truncated. Testing well
// under the cap leaves `>` vs `>=` undetected.
func TestRunEndpointScan_ExactlyAtCapReportsNoTruncation(t *testing.T) {
	t.Parallel()

	urls := make([]string, 0, DefaultMaxEndpointsPerRecord)
	for i := range DefaultMaxEndpointsPerRecord {
		urls = append(urls, fmt.Sprintf("https://mcp%d.example.com/mcp", i))
	}

	r := NewMCPRunner(MCPConfig{CLIPath: fakeCLIPath(t)})
	got := r.runEndpointScan(context.Background(), recordWithEndpoints(t, urls...))

	if len(got.Notices) != 0 {
		t.Errorf("a record exactly at the cap is fully scanned, so it must not report truncation: %v", got.Notices)
	}

	wantFindings := DefaultMaxEndpointsPerRecord * len(endpointSubcommands)
	if len(got.Findings) != wantFindings {
		t.Errorf("want %d findings at the cap, got %d", wantFindings, len(got.Findings))
	}
}

// The cap only bounds amplification if its value stays small. Both sides of
// the count assertions above are the constant itself, so nothing else here
// would notice the default being raised to 500 - which is the exact number the
// change exists to prevent. This pins the magnitude, so raising it has to be a
// deliberate argument against the reasoning rather than a quiet edit.
func TestDefaultMaxEndpointsPerRecord_StaysSmall(t *testing.T) {
	t.Parallel()

	const upperBound = 16

	if DefaultMaxEndpointsPerRecord > upperBound {
		t.Fatalf("default cap is %d, above the %d ceiling: each endpoint costs %d scanner invocations at a publisher-chosen host, so a large default reinstates the amplification this cap exists to stop",
			DefaultMaxEndpointsPerRecord, upperBound, len(endpointSubcommands))
	}
}

// Duplicate URLs must not consume cap budget or multiply requests to one host.
func TestExtractEndpoints_DeduplicatesURLs(t *testing.T) {
	t.Parallel()

	const dupe = "https://victim.example.com/mcp"

	r := NewMCPRunner(MCPConfig{CLIPath: fakeCLIPath(t)})
	rec := recordWithEndpoints(t, dupe, dupe, dupe, dupe, dupe, dupe, dupe, dupe)

	got := r.runEndpointScan(context.Background(), rec)

	// One distinct endpoint x 4 subcommands. Without dedup this is 32
	// requests at a single host, all under the cap and therefore silent.
	const wantFindings = 4

	if len(got.Findings) != wantFindings {
		t.Errorf("eight copies of one URL should scan once: want %d findings, got %d", wantFindings, len(got.Findings))
	}
}

// The notice has to survive the merge inside runEndpointScan AND the merge in
// Run that combines the endpoint and source phases. The first implementation
// wrote it to SkippedReason, which merge drops unless every input skipped, so
// it vanished exactly when one endpoint scanned successfully.
func TestRun_TruncationNoticeSurvivesBothMerges(t *testing.T) {
	t.Parallel()

	urls := make([]string, 0, DefaultMaxEndpointsPerRecord+4)
	for i := range DefaultMaxEndpointsPerRecord + 4 {
		urls = append(urls, fmt.Sprintf("https://mcp%d.example.com/mcp", i))
	}

	r := NewMCPRunner(MCPConfig{CLIPath: fakeCLIPath(t)})

	got, err := r.Run(context.Background(), recordWithEndpoints(t, urls...))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(got.Findings) == 0 {
		t.Fatal("test invalid: endpoints must scan successfully, or the notice would survive for the wrong reason")
	}

	var found bool

	for _, n := range got.Notices {
		if strings.Contains(n, "scanned the first") {
			found = true
		}
	}

	if !found {
		t.Errorf("truncation notice did not survive Run: %v", got.Notices)
	}
}

// AllowPrivateEndpoints has to reach the scan path, not just the validator.
func TestRunEndpointScan_AllowPrivate_ScansPrivateEndpoint(t *testing.T) {
	t.Parallel()

	r := NewMCPRunner(MCPConfig{CLIPath: fakeCLIPath(t), AllowPrivateEndpoints: true})
	rec := recordWithEndpoints(t, "https://10.0.0.5/mcp")

	got := r.runEndpointScan(context.Background(), rec)

	if got.Skipped {
		t.Fatalf("private endpoint should be scanned when allowed, got skipped: %s", got.SkippedReason)
	}

	if len(got.Findings) == 0 {
		t.Error("want findings from the private endpoint scan, got none")
	}
}

// Every case here was accepted by an earlier revision of validateEndpointURL
// and was found by an adversarial review of this PR. They are grouped
// separately so a future refactor that reintroduces "if it does not parse as
// an IP it must be a hostname" fails loudly and specifically.
func TestValidateEndpointURL_RejectedBypasses(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		url  string
		why  string
	}{
		{"decimal metadata address", "https://2852039166/", "resolves to 169.254.169.254"},
		{"decimal loopback", "https://2130706433/", "resolves to 127.0.0.1"},
		{"short-form loopback", "https://127.1/", "resolves to 127.0.0.1"},
		{"octal loopback", "https://0177.0.0.1/", "resolves to 127.0.0.1"},
		{"hex loopback", "https://0x7f000001/", "resolves to 127.0.0.1"},
		{"zero-padded loopback", "https://127.000.000.001/", "resolves to 127.0.0.1"},
		{"bare zero", "https://0/", "resolves to 0.0.0.0"},
		{"trailing-dot loopback", "https://127.0.0.1./", "trailing dot must not skip the range check"},
		{"trailing-dot metadata", "https://169.254.169.254./", "trailing dot must not skip the range check"},
		{"trailing-dot private", "https://10.0.0.1./", "trailing dot must not skip the range check"},
		{"trailing-dot dotted localhost", "https://foo.localhost./", "trailing dot must not skip the loopback name check"},
		{"alibaba metadata", "https://100.100.100.200/", "cloud metadata service outside RFC 1918"},
		{"cgnat", "https://100.64.0.1/", "RFC 6598, used as pod/node CIDR by managed Kubernetes"},
		{"ietf protocol assignments", "https://192.0.0.1/", "reserved"},
		{"benchmarking range", "https://198.18.0.1/", "RFC 2544"},
		{"test-net-1", "https://192.0.2.5/", "reserved for documentation"},
		{"test-net-3", "https://203.0.113.5/", "reserved for documentation"},
		{"reserved class e", "https://240.0.0.1/", "reserved"},
		{"broadcast", "https://255.255.255.255/", "limited broadcast"},
		{"ipv4-compatible ipv6 loopback", "https://[::127.0.0.1]/", "embeds 127.0.0.1"},
		{"nat64 loopback", "https://[64:ff9b::7f00:1]/", "NAT64 embedding 127.0.0.1"},
		{"6to4 loopback", "https://[2002:7f00:1::]/", "6to4 embedding 127.0.0.1"},
		{"zoned loopback", "https://[::1%25lo]/", "interface zone must not skip the range check"},
		{"zoned link-local", "https://[fe80::1%25eth0]/", "interface zone must not skip the range check"},
		{"zoned unique-local", "https://[fc00::1%25x]/", "interface zone must not skip the range check"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if err := validateEndpointURL(tc.url, false, false); err == nil {
				t.Errorf("validateEndpointURL(%q) = nil, want rejection: %s", tc.url, tc.why)
			}
		})
	}
}

// Hostnames that merely look numeric-adjacent must still work, or the
// all-numeric rule would be over-broad and break real deployments.
func TestValidateEndpointURL_AcceptsLegitimateHostnames(t *testing.T) {
	t.Parallel()

	cases := []string{
		"https://mcp1.example.com/mcp",
		"https://1mcp.example.com/mcp",
		"https://v2.api.example.com/mcp",
		"https://xn--80ak6aa92e.example.com/mcp",
		"https://a-b-c.example.com/mcp",
		"https://example.com./mcp",
		"https://0x-not-hex.example.com/mcp",
	}

	for _, u := range cases {
		t.Run(u, func(t *testing.T) {
			t.Parallel()

			if err := validateEndpointURL(u, false, false); err != nil {
				t.Errorf("validateEndpointURL(%q) = %v, want accepted", u, err)
			}
		})
	}
}

// AllowInsecureTransport is deliberately independent of AllowPrivateEndpoints:
// needing to reach a private-range endpoint must not silently also permit
// cleartext scans of public hosts.
func TestValidateEndpointURL_InsecureTransportIsSeparateFromPrivate(t *testing.T) {
	t.Parallel()

	const publicHTTP = "http://mcp.example.com/mcp"

	if err := validateEndpointURL(publicHTTP, true, false); err == nil {
		t.Error("allowPrivate alone must not permit cleartext http to a public host")
	}

	if err := validateEndpointURL(publicHTTP, false, true); err != nil {
		t.Errorf("allowInsecure should permit http: %v", err)
	}

	if err := validateEndpointURL("https://10.0.0.5/mcp", false, true); err == nil {
		t.Error("allowInsecure alone must not permit a private-range endpoint")
	}
}
