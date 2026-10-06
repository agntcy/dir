// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package ansresolver

import (
	"context"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/agntcy/dir/client/utils/identity/resolvers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseBadgeRecord(t *testing.T) {
	tests := []struct {
		name        string
		txt         string
		wantURL     string
		wantVersion string
		wantOK      bool
	}{
		{name: "versioned", txt: "v=ans-badge1; version=v1.0.0; url=" + testBadgeURL, wantURL: testBadgeURL, wantVersion: "v1.0.0", wantOK: true},
		{name: "versionless", txt: "v=ans-badge1; url=" + testBadgeURL, wantURL: testBadgeURL, wantOK: true},
		{name: "no spaces", txt: "v=ans-badge1;version=v2.1.0;url=" + testBadgeURL, wantURL: testBadgeURL, wantVersion: "v2.1.0", wantOK: true},
		{name: "unknown fields are ignored", txt: "v=ans-badge1; foo=bar; url=" + testBadgeURL + "; junk", wantURL: testBadgeURL, wantOK: true},
		{name: "other format", txt: "v=ra-badge1; url=" + testBadgeURL},
		{name: "unrelated record", txt: "v=spf1 include:_spf.example.com ~all"},
		{name: "missing url", txt: "v=ans-badge1; version=v1.0.0"},
		{name: "missing format", txt: "url=" + testBadgeURL},
		{name: "unparsable version", txt: "v=ans-badge1; version=latest; url=" + testBadgeURL},
		{name: "empty", txt: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			record, ok := parseBadgeRecord(tt.txt)
			assert.Equal(t, tt.wantOK, ok)

			if !tt.wantOK {
				return
			}

			assert.Equal(t, tt.wantURL, record.url)

			if tt.wantVersion == "" {
				assert.Nil(t, record.version)
			} else {
				require.NotNil(t, record.version)
				assert.Equal(t, tt.wantVersion, record.version.String())
			}
		})
	}
}

func TestParseBadgeURL(t *testing.T) {
	const id = testAgentID

	trusted := map[string]struct{}{testLogHost: {}, "localhost:18081": {}, "[::1]:18081": {}}
	canonical := TrustedLog{Origin: testLogOrigin, AgentID: id}

	tests := []struct {
		name    string
		raw     string
		want    TrustedLog
		wantErr string
	}{
		{name: "canonical", raw: testLogOrigin + "/v1/agents/" + id, want: canonical},
		{name: "explicit port", raw: "https://localhost:18081/v1/agents/" + id, want: TrustedLog{Origin: "https://localhost:18081", AgentID: id}},
		{name: "default port dropped", raw: "https://log.example.com:443/v1/agents/" + id, want: canonical},
		{name: "host lowercased", raw: "https://LOG.Example.COM/v1/agents/" + id, want: canonical},
		{name: "ipv6 host", raw: "https://[::1]:18081/v1/agents/" + id, want: TrustedLog{Origin: "https://[::1]:18081", AgentID: id}},
		{name: "untrusted host", raw: "https://other.example.com/v1/agents/" + id, wantErr: `log host "other.example.com" is not a trusted transparency log`},
		{name: "untrusted port", raw: "https://log.example.com:8443/v1/agents/" + id, wantErr: "is not a trusted transparency log"},
		{name: "uppercase agent id", raw: testLogOrigin + "/v1/agents/" + strings.ToUpper(id), wantErr: "agent id is not a lowercase UUID"},
		{name: "http scheme", raw: "http://log.example.com/v1/agents/" + id, wantErr: `scheme "http" is not https`},
		{name: "empty", raw: "", wantErr: "scheme"},
		{name: "userinfo", raw: "https://user:secret@log.example.com/v1/agents/" + id, wantErr: "userinfo"},
		{name: "query", raw: testLogOrigin + "/v1/agents/" + id + "?x=1", wantErr: "query"},
		{name: "empty query", raw: testLogOrigin + "/v1/agents/" + id + "?", wantErr: "query"},
		{name: "fragment", raw: testLogOrigin + "/v1/agents/" + id + "#frag", wantErr: "fragment"},
		{name: "trailing slash", raw: testLogOrigin + "/v1/agents/" + id + "/", wantErr: "path"},
		{name: "extra segment", raw: testLogOrigin + "/v1/agents/" + id + "/status-token", wantErr: "path"},
		{name: "missing agent id", raw: testLogOrigin + "/v1/agents", wantErr: "path"},
		{name: "wrong prefix", raw: testLogOrigin + "/v2/agents/" + id, wantErr: "path"},
		{name: "dot segment", raw: testLogOrigin + "/v1/agents/../" + id, wantErr: "path"},
		{name: "empty segment", raw: testLogOrigin + "//v1/agents/" + id, wantErr: "path"},
		{name: "percent-encoded prefix", raw: testLogOrigin + "/%761/agents/" + id, wantErr: "path"},
		{name: "percent-encoded agent id", raw: testLogOrigin + "/v1/agents/%35" + id[1:], wantErr: "agent id"},
		{name: "not a uuid", raw: testLogOrigin + "/v1/agents/not-a-uuid", wantErr: "agent id"},
		{name: "no host", raw: "https:///v1/agents/" + id, wantErr: "host"},
		{name: "port without a host name", raw: "https://:443/v1/agents/" + id, wantErr: "host is malformed"},
		{name: "not a url", raw: "https://log.example.com:abc/v1/agents/" + id, wantErr: "not a valid URL"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseBadgeURL(tt.raw, trusted)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				assert.True(t, strings.HasPrefix(err.Error(), "ans badge: "), "error %q lacks the stage prefix", err)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseBadgeURLDoesNotEchoTheWholeScheme(t *testing.T) {
	long := strings.Repeat("x", maxHostLength+40)

	_, err := parseBadgeURL(long+"://log.example.com/v1/agents/"+testAgentID, map[string]struct{}{})
	require.Error(t, err)
	assert.NotContains(t, err.Error(), long)
}

func TestLookupBadge(t *testing.T) {
	other := "https://log.example.com/v1/agents/" + otherAgentID

	tests := []struct {
		name    string
		records []string
		dnsErr  error
		want    TrustedLog
		wantErr string
	}{
		{name: "exact version", records: []string{badgeTXT("v1.0.0", testBadgeURL)}, want: TrustedLog{Origin: testLogOrigin, AgentID: testAgentID}},
		{name: "exact version wins over versionless", records: []string{badgeTXT("v2.0.0", other), "v=ans-badge1; url=" + other, badgeTXT("v1.0.0", testBadgeURL)}, want: TrustedLog{Origin: testLogOrigin, AgentID: testAgentID}},
		{name: "versionless fallback", records: []string{badgeTXT("v2.0.0", other), "v=ans-badge1; url=" + testBadgeURL}, want: TrustedLog{Origin: testLogOrigin, AgentID: testAgentID}},
		{name: "unrelated records are skipped", records: []string{"v=spf1 -all", badgeTXT("v1.0.0", testBadgeURL)}, want: TrustedLog{Origin: testLogOrigin, AgentID: testAgentID}},
		{name: "no record", records: []string{"v=spf1 -all"}, wantErr: "ans badge: no _ans-badge.agent.example.com record names version v1.0.0"},
		{name: "other version only", records: []string{badgeTXT("v2.0.0", testBadgeURL)}, wantErr: "no _ans-badge.agent.example.com record names version v1.0.0"},
		{name: "two exact matches", records: []string{badgeTXT("v1.0.0", testBadgeURL), badgeTXT("v1.0.0", other)}, wantErr: "ans badge: 2 _ans-badge.agent.example.com records name version v1.0.0; expected one"},
		{name: "two versionless records", records: []string{"v=ans-badge1; url=" + testBadgeURL, "v=ans-badge1; url=" + other}, wantErr: "2 _ans-badge.agent.example.com records name version v1.0.0; expected one"},
		{name: "untrusted log", records: []string{badgeTXT("v1.0.0", "https://other.example.com/v1/agents/"+testAgentID)}, wantErr: "is not a trusted transparency log"},
		{name: "name does not exist", dnsErr: &net.DNSError{Err: "no such host", Name: testBadgeName, IsNotFound: true}, wantErr: "ans badge: lookup _ans-badge.agent.example.com: lookup _ans-badge.agent.example.com: no such host"},
		{name: "server failure", dnsErr: &net.DNSError{Err: "server misbehaving", Name: testBadgeName}, wantErr: "ans badge: lookup _ans-badge.agent.example.com: "},
		{name: "timeout", dnsErr: dnsError(testBadgeName, context.DeadlineExceeded), wantErr: "ans badge: lookup _ans-badge.agent.example.com: "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dns := newFakeTXT(tt.records...)
			dns.err = tt.dnsErr

			name, err := parseAgentName(testAnsName)
			require.NoError(t, err)

			got, err := newTestResolver(t, unpinnedConfig(), dns, &fakeLog{}).lookupBadge(t.Context(), name)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)

				// The zone is the publisher's: every failure is final whatever its cause,
				// and the cause stays readable.
				require.ErrorIs(t, err, resolvers.ErrFinal)

				if tt.dnsErr != nil {
					var dnsErr *net.DNSError

					require.ErrorAs(t, err, &dnsErr)
				}

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestLookupBadgeUnderADoneContext(t *testing.T) {
	dns := newFakeTXT(badgeTXT(testVersion, testBadgeURL))
	r := newTestResolver(t, unpinnedConfig(), dns, &fakeLog{})

	name, err := parseAgentName(testAnsName)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(t.Context(), time.Nanosecond)
	defer cancel()

	<-ctx.Done()

	_, err = r.lookupBadge(ctx, name)
	require.ErrorContains(t, err, "ans badge: lookup")
	require.ErrorIs(t, err, resolvers.ErrFinal)
}

func TestTruncate(t *testing.T) {
	long := strings.Repeat("a", maxHostLength+1)

	tests := []struct {
		name string
		text string
		want string
	}{
		{name: "short text is kept", text: testLogHost, want: testLogHost},
		{name: "text at the limit is kept", text: long[:maxHostLength], want: long[:maxHostLength]},
		{name: "longer text is cut", text: long, want: long[:maxHostLength]},
		{name: "empty", text: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, truncate(tt.text))
		})
	}
}
