// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package verification

import (
	"encoding/base64"
	"testing"
)

func TestParseDNSTXTRecord(t *testing.T) {
	tests := []struct {
		name      string
		record    string
		wantType  string
		wantError bool
	}{
		{
			name:     "valid ed25519 key",
			record:   "v=oasf1; k=ed25519; p=" + base64.StdEncoding.EncodeToString([]byte("test-public-key")),
			wantType: "ed25519",
		},
		{
			name:     "valid ecdsa-p256 key",
			record:   "v=oasf1; k=ecdsa-p256; p=" + base64.StdEncoding.EncodeToString([]byte("test-public-key")),
			wantType: "ecdsa-p256",
		},
		{
			name:      "missing version",
			record:    "k=ed25519; p=" + base64.StdEncoding.EncodeToString([]byte("test")),
			wantError: true,
		},
		{
			name:      "wrong version",
			record:    "v=oasf2; k=ed25519; p=" + base64.StdEncoding.EncodeToString([]byte("test")),
			wantError: true,
		},
		{
			name:      "missing key type",
			record:    "v=oasf1; p=" + base64.StdEncoding.EncodeToString([]byte("test")),
			wantError: true,
		},
		{
			name:      "unsupported key type",
			record:    "v=oasf1; k=dsa; p=" + base64.StdEncoding.EncodeToString([]byte("test")),
			wantError: true,
		},
		{
			name:      "missing public key",
			record:    "v=oasf1; k=ed25519",
			wantError: true,
		},
		{
			name:      "invalid base64",
			record:    "v=oasf1; k=ed25519; p=not-valid-base64!!!",
			wantError: true,
		},
		{
			name:     "extra whitespace",
			record:   "  v=oasf1 ;  k=ed25519 ;  p=" + base64.StdEncoding.EncodeToString([]byte("test"))+ "  ",
			wantType: "ed25519",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, err := ParseDNSTXTRecord(tt.record)

			if tt.wantError {
				if err == nil {
					t.Errorf("ParseDNSTXTRecord() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("ParseDNSTXTRecord() unexpected error: %v", err)
				return
			}

			if key.Type != tt.wantType {
				t.Errorf("ParseDNSTXTRecord() type = %v, want %v", key.Type, tt.wantType)
			}
		})
	}
}

func TestParseWellKnownKey(t *testing.T) {
	tests := []struct {
		name      string
		key       WellKnownKey
		wantID    string
		wantType  string
		wantError bool
	}{
		{
			name: "valid key with ID",
			key: WellKnownKey{
				ID:        "key-1",
				Type:      "ed25519",
				PublicKey: base64.StdEncoding.EncodeToString([]byte("test-public-key")),
			},
			wantID:   "key-1",
			wantType: "ed25519",
		},
		{
			name: "valid key without ID",
			key: WellKnownKey{
				Type:      "ecdsa-p256",
				PublicKey: base64.StdEncoding.EncodeToString([]byte("test-public-key")),
			},
			wantID:   "",
			wantType: "ecdsa-p256",
		},
		{
			name: "unsupported key type",
			key: WellKnownKey{
				Type:      "dsa",
				PublicKey: base64.StdEncoding.EncodeToString([]byte("test")),
			},
			wantError: true,
		},
		{
			name: "invalid base64",
			key: WellKnownKey{
				Type:      "ed25519",
				PublicKey: "not-valid-base64!!!",
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, err := ParseWellKnownKey(tt.key)

			if tt.wantError {
				if err == nil {
					t.Errorf("ParseWellKnownKey() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("ParseWellKnownKey() unexpected error: %v", err)
				return
			}

			if key.ID != tt.wantID {
				t.Errorf("ParseWellKnownKey() ID = %v, want %v", key.ID, tt.wantID)
			}

			if key.Type != tt.wantType {
				t.Errorf("ParseWellKnownKey() type = %v, want %v", key.Type, tt.wantType)
			}
		})
	}
}

func TestMatchKey(t *testing.T) {
	key1 := []byte("public-key-1")
	key2 := []byte("public-key-2")
	key3 := []byte("public-key-3")

	domainKeys := []PublicKey{
		{ID: "key-1", Type: "ed25519", Key: key1},
		{ID: "key-2", Type: "ed25519", Key: key2},
	}

	tests := []struct {
		name       string
		signingKey []byte
		wantMatch  bool
		wantID     string
	}{
		{
			name:       "matches first key",
			signingKey: key1,
			wantMatch:  true,
			wantID:     "key-1",
		},
		{
			name:       "matches second key",
			signingKey: key2,
			wantMatch:  true,
			wantID:     "key-2",
		},
		{
			name:       "no match",
			signingKey: key3,
			wantMatch:  false,
		},
		{
			name:       "empty signing key",
			signingKey: []byte{},
			wantMatch:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matched, found := MatchKey(tt.signingKey, domainKeys)

			if found != tt.wantMatch {
				t.Errorf("MatchKey() found = %v, want %v", found, tt.wantMatch)
			}

			if found && matched.ID != tt.wantID {
				t.Errorf("MatchKey() matched ID = %v, want %v", matched.ID, tt.wantID)
			}
		})
	}
}
