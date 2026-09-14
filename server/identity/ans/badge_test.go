// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package ans

import (
	"strings"
	"testing"

	"github.com/agentnameservice/ans-sdk-go/verify"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseBadgeURL(t *testing.T) {
	const id = testAgentID

	canonical := badgeTarget{LogBase: "https://log.example.com", LogHost: "log.example.com", AgentID: id}

	tests := []struct {
		name    string
		raw     string
		want    badgeTarget
		wantErr string
	}{
		{name: "canonical", raw: "https://log.example.com/v1/agents/" + id, want: canonical},
		{name: "explicit port", raw: "https://localhost:18081/v1/agents/" + id, want: badgeTarget{LogBase: "https://localhost:18081", LogHost: "localhost:18081", AgentID: id}},
		{name: "default port dropped", raw: "https://log.example.com:443/v1/agents/" + id, want: canonical},
		{name: "host lowercased", raw: "https://LOG.Example.COM/v1/agents/" + id, want: canonical},
		{name: "agent id lowercased", raw: "https://log.example.com/v1/agents/" + strings.ToUpper(id), want: canonical},
		{name: "ipv6 host", raw: "https://[::1]:18081/v1/agents/" + id, want: badgeTarget{LogBase: "https://[::1]:18081", LogHost: "[::1]:18081", AgentID: id}},
		{name: "http scheme", raw: "http://log.example.com/v1/agents/" + id, wantErr: "scheme is not https"},
		{name: "empty", raw: "", wantErr: "scheme"},
		{name: "userinfo", raw: "https://user:secret@log.example.com/v1/agents/" + id, wantErr: "userinfo"},
		{name: "query", raw: "https://log.example.com/v1/agents/" + id + "?x=1", wantErr: "query"},
		{name: "empty query", raw: "https://log.example.com/v1/agents/" + id + "?", wantErr: "query"},
		{name: "fragment", raw: "https://log.example.com/v1/agents/" + id + "#frag", wantErr: "fragment"},
		{name: "trailing slash", raw: "https://log.example.com/v1/agents/" + id + "/", wantErr: "path"},
		{name: "extra segment", raw: "https://log.example.com/v1/agents/" + id + "/status-token", wantErr: "path"},
		{name: "missing agent id", raw: "https://log.example.com/v1/agents", wantErr: "path"},
		{name: "wrong prefix", raw: "https://log.example.com/v2/agents/" + id, wantErr: "path"},
		{name: "dot segment", raw: "https://log.example.com/v1/agents/../" + id, wantErr: "path"},
		{name: "empty segment", raw: "https://log.example.com//v1/agents/" + id, wantErr: "path"},
		{name: "percent-encoded prefix", raw: "https://log.example.com/%761/agents/" + id, wantErr: "path"},
		{name: "percent-encoded agent id", raw: "https://log.example.com/v1/agents/%35" + id[1:], wantErr: "agent id"},
		{name: "not a uuid", raw: "https://log.example.com/v1/agents/not-a-uuid", wantErr: "agent id"},
		{name: "no host", raw: "https:///v1/agents/" + id, wantErr: "host"},
		{name: "port without a host name", raw: "https://:443/v1/agents/" + id, wantErr: "host is malformed"},
		{name: "not a url", raw: "https://log.example.com:abc/v1/agents/" + id, wantErr: "not a valid URL"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseBadgeURL(tt.raw)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				assert.True(t, strings.HasPrefix(err.Error(), "ans badge-url: "), "error %q lacks the stage prefix", err)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseBadgeURLDoesNotEchoTheScheme(t *testing.T) {
	_, err := parseBadgeURL("javascript-and-a-very-long-scheme://log.example.com/v1/agents/" + testAgentID)
	require.EqualError(t, err, "ans badge-url: badge URL scheme is not https")
}

func TestBadgeSourceName(t *testing.T) {
	assert.Equal(t, "_ans-badge", badgeSourceName(verify.BadgeRecordSourceAnsBadge))
	assert.Equal(t, "_ra-badge", badgeSourceName(verify.BadgeRecordSourceRaBadge))
}

func TestTruncateHost(t *testing.T) {
	long := strings.Repeat("a", maxHostLength+1)

	tests := []struct {
		name string
		host string
		want string
	}{
		{name: "short host is kept", host: testLogHost, want: testLogHost},
		{name: "host at the limit is kept", host: long[:maxHostLength], want: long[:maxHostLength]},
		{name: "longer host is cut", host: long, want: long[:maxHostLength]},
		{name: "empty", host: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, truncateHost(tt.host))
		})
	}
}
