// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package ans

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseAgentName(t *testing.T) {
	const (
		notAnsName   = "ans name: subject is not an ANS name"
		notCanonical = "ans name: subject is not in canonical form"
	)

	tests := []struct {
		name    string
		subject string
		wantErr string
	}{
		{name: "canonical", subject: testAnsName},
		{name: "multi-digit version", subject: "ans://v10.2.33.agent.example.com"},
		{name: "single-label host", subject: "ans://v1.0.0.localhost"},
		{name: "uppercase host is not canonical", subject: "ans://v1.0.0.Agent.Example.COM", wantErr: notCanonical},
		{name: "leading zero in the version is not canonical", subject: "ans://v01.0.0.agent.example.com", wantErr: notCanonical},
		{name: "trailing dot is not canonical", subject: testAnsName + ".", wantErr: notCanonical},
		{name: "uppercase scheme", subject: "ANS://v1.0.0.agent.example.com", wantErr: notAnsName},
		{name: "version without v", subject: "ans://1.0.0.agent.example.com", wantErr: notAnsName},
		{name: "two-part version", subject: "ans://v1.0.agent.example.com", wantErr: notAnsName},
		{name: "version label outside uint32", subject: "ans://v4294967296.0.0.agent.example.com", wantErr: notAnsName},
		{name: "dns subject", subject: "dns:agent.example.com", wantErr: notAnsName},
		{name: "empty subject", subject: "", wantErr: notAnsName},
		{name: "empty host", subject: "ans://v1.0.0.", wantErr: `ans name: host "" is not a valid DNS name`},
		{name: "underscore in the host", subject: "ans://v1.0.0.bad_host.example.com", wantErr: `ans name: host "bad_host.example.com" is not a valid DNS name`},
		{name: "label starting with a hyphen", subject: "ans://v1.0.0.-agent.example.com", wantErr: `ans name: host "-agent.example.com" is not a valid DNS name`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseAgentName(tt.subject)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.subject, got.String())
			assert.Equal(t, got.host, got.fqdn.String())
		})
	}
}

func TestParseAgentNameTruncatesTheHostItEchoes(t *testing.T) {
	host := strings.Repeat("a", maxEchoLength+50)

	_, err := parseAgentName(ansScheme + testVersion + "." + host)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `host "`+host[:maxEchoLength]+`"`)
	assert.NotContains(t, err.Error(), host)
}

func TestAgentNameMatches(t *testing.T) {
	want, err := parseAgentName(testAnsName)
	require.NoError(t, err)

	tests := []struct {
		name string
		raw  string
		want bool
	}{
		{name: "same spelling", raw: testAnsName, want: true},
		{name: "uppercase host", raw: "ans://v1.0.0.AGENT.Example.com", want: true},
		{name: "leading zero in the version", raw: "ans://v01.0.0.agent.example.com", want: true},
		{name: "other version", raw: "ans://v1.0.1.agent.example.com"},
		{name: "other host", raw: "ans://v1.0.0.other.example.com"},
		{name: "trailing dot", raw: testAnsName + "."},
		{name: "not an ans name", raw: "spiffe://trust.example.com/agent"},
		{name: "empty", raw: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, want.matches(tt.raw))
		})
	}
}
