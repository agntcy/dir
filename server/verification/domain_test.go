// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package verification

import (
	"testing"
)

func TestExtractDomain(t *testing.T) {
	tests := []struct {
		name       string
		recordName string
		want       string
	}{
		// Valid formats
		{
			name:       "domain only",
			recordName: "cisco.com",
			want:       "cisco.com",
		},
		{
			name:       "domain with path",
			recordName: "cisco.com/marketing-agent",
			want:       "cisco.com",
		},
		{
			name:       "domain with nested path",
			recordName: "example.org/agents/v1/helper",
			want:       "example.org",
		},
		{
			name:       "subdomain",
			recordName: "agents.cisco.com/marketing",
			want:       "agents.cisco.com",
		},
		{
			name:       "subdomain without path",
			recordName: "agents.cisco.com",
			want:       "agents.cisco.com",
		},
		{
			name:       "co.uk domain",
			recordName: "example.co.uk/agent",
			want:       "example.co.uk",
		},

		// Invalid formats
		{
			name:       "empty string",
			recordName: "",
			want:       "",
		},
		{
			name:       "no dot",
			recordName: "localhost",
			want:       "",
		},
		{
			name:       "no dot with path",
			recordName: "localhost/agent",
			want:       "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractDomain(tt.recordName)
			if got != tt.want {
				t.Errorf("ExtractDomain(%q) = %q, want %q", tt.recordName, got, tt.want)
			}
		})
	}
}
