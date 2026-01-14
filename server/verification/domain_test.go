// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package verification

import (
	"testing"
)

func TestParseName(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		wantProtocol string
		wantDomain   string
		wantPath     string
		wantFullName string
		wantNil      bool
	}{
		// Valid formats without protocol prefix
		{
			name:         "domain only",
			input:        "cisco.com",
			wantProtocol: "",
			wantDomain:   "cisco.com",
			wantPath:     "",
			wantFullName: "cisco.com",
		},
		{
			name:         "domain with path",
			input:        "cisco.com/marketing-agent",
			wantProtocol: "",
			wantDomain:   "cisco.com",
			wantPath:     "marketing-agent",
			wantFullName: "cisco.com/marketing-agent",
		},
		{
			name:         "domain with nested path",
			input:        "example.org/agents/v1/helper",
			wantProtocol: "",
			wantDomain:   "example.org",
			wantPath:     "agents/v1/helper",
			wantFullName: "example.org/agents/v1/helper",
		},
		{
			name:         "subdomain",
			input:        "agents.cisco.com/marketing",
			wantProtocol: "",
			wantDomain:   "agents.cisco.com",
			wantPath:     "marketing",
			wantFullName: "agents.cisco.com/marketing",
		},

		// Valid formats with dns:// prefix
		{
			name:         "dns protocol domain only",
			input:        "dns://cisco.com",
			wantProtocol: DNSProtocol,
			wantDomain:   "cisco.com",
			wantPath:     "",
			wantFullName: "cisco.com",
		},
		{
			name:         "dns protocol with path",
			input:        "dns://cisco.com/agent",
			wantProtocol: DNSProtocol,
			wantDomain:   "cisco.com",
			wantPath:     "agent",
			wantFullName: "cisco.com/agent",
		},

		// Valid formats with wellknown:// prefix
		{
			name:         "wellknown protocol domain only",
			input:        "wellknown://example.org",
			wantProtocol: WellKnownProtocol,
			wantDomain:   "example.org",
			wantPath:     "",
			wantFullName: "example.org",
		},
		{
			name:         "wellknown protocol with path",
			input:        "wellknown://example.org/my-agent",
			wantProtocol: WellKnownProtocol,
			wantDomain:   "example.org",
			wantPath:     "my-agent",
			wantFullName: "example.org/my-agent",
		},

		// Invalid formats
		{
			name:    "empty string",
			input:   "",
			wantNil: true,
		},
		{
			name:    "no dot",
			input:   "localhost",
			wantNil: true,
		},
		{
			name:    "no dot with path",
			input:   "localhost/agent",
			wantNil: true,
		},
		{
			name:    "dns protocol no dot",
			input:   "dns://localhost/agent",
			wantNil: true,
		},
		{
			name:    "wellknown protocol no dot",
			input:   "wellknown://localhost",
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseName(tt.input)

			if tt.wantNil {
				if got != nil {
					t.Errorf("ParseName(%q) = %+v, want nil", tt.input, got)
				}

				return
			}

			if got == nil {
				t.Errorf("ParseName(%q) = nil, want non-nil", tt.input)

				return
			}

			if got.Protocol != tt.wantProtocol {
				t.Errorf("ParseName(%q).Protocol = %q, want %q", tt.input, got.Protocol, tt.wantProtocol)
			}

			if got.Domain != tt.wantDomain {
				t.Errorf("ParseName(%q).Domain = %q, want %q", tt.input, got.Domain, tt.wantDomain)
			}

			if got.Path != tt.wantPath {
				t.Errorf("ParseName(%q).Path = %q, want %q", tt.input, got.Path, tt.wantPath)
			}

			if got.FullName != tt.wantFullName {
				t.Errorf("ParseName(%q).FullName = %q, want %q", tt.input, got.FullName, tt.wantFullName)
			}
		})
	}
}

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
		// With protocol prefix
		{
			name:       "dns protocol",
			recordName: "dns://cisco.com/agent",
			want:       "cisco.com",
		},
		{
			name:       "wellknown protocol",
			recordName: "wellknown://example.org/agent",
			want:       "example.org",
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
