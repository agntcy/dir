// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package scanner

import (
	"context"
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

			if err := validateEndpointURL(tc.url, false); err == nil {
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

			if err := validateEndpointURL(u, false); err != nil {
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

			if err := validateEndpointURL(u, true); err != nil {
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
		if err := validateEndpointURL(u, true); err == nil {
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
