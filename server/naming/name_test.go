// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package naming

import (
	"testing"
)

func TestParseName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    *ParsedName
		wantNil bool
	}{
		{
			name:  "simple domain",
			input: "cisco.com",
			want: &ParsedName{
				Domain:   "cisco.com",
				FullName: "cisco.com",
			},
		},
		{
			name:  "domain with path",
			input: "cisco.com/agent",
			want: &ParsedName{
				Domain:   "cisco.com",
				Path:     "agent",
				FullName: "cisco.com/agent",
			},
		},
		{
			name:  "https protocol",
			input: "https://example.org/test",
			want: &ParsedName{
				Protocol: HTTPSProtocol,
				Domain:   "example.org",
				Path:     "test",
				FullName: "example.org/test",
			},
		},
		{
			name:  "http protocol with port",
			input: "http://localhost:8080/agent",
			want: &ParsedName{
				Protocol: HTTPProtocol,
				Domain:   "localhost:8080",
				Path:     "agent",
				FullName: "localhost:8080/agent",
			},
		},
		{
			name:    "empty string",
			input:   "",
			wantNil: true,
		},
		{
			name:    "no dot in domain and not localhost",
			input:   "invalid",
			wantNil: true,
		},
		{
			name:  "localhost with port is valid",
			input: "http://localhost:8080/agent",
			want: &ParsedName{
				Protocol: HTTPProtocol,
				Domain:   "localhost:8080",
				Path:     "agent",
				FullName: "localhost:8080/agent",
			},
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
				t.Fatalf("ParseName(%q) = nil, want %+v", tt.input, tt.want)
			}

			if got.Protocol != tt.want.Protocol {
				t.Errorf("ParseName(%q).Protocol = %q, want %q", tt.input, got.Protocol, tt.want.Protocol)
			}

			if got.Domain != tt.want.Domain {
				t.Errorf("ParseName(%q).Domain = %q, want %q", tt.input, got.Domain, tt.want.Domain)
			}

			if got.Path != tt.want.Path {
				t.Errorf("ParseName(%q).Path = %q, want %q", tt.input, got.Path, tt.want.Path)
			}

			if got.FullName != tt.want.FullName {
				t.Errorf("ParseName(%q).FullName = %q, want %q", tt.input, got.FullName, tt.want.FullName)
			}
		})
	}
}

func TestExtractDomain(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "simple domain",
			input: "cisco.com",
			want:  "cisco.com",
		},
		{
			name:  "domain with path",
			input: "cisco.com/agent",
			want:  "cisco.com",
		},
		{
			name:  "https protocol",
			input: "https://example.org",
			want:  "example.org",
		},
		{
			name:  "http protocol with localhost",
			input: "http://localhost:8080/agent",
			want:  "localhost:8080",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "no dot and not localhost",
			input: "invalid",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractDomain(tt.input)
			if got != tt.want {
				t.Errorf("ExtractDomain(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
