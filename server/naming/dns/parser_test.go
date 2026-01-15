// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package dns

import (
	"testing"
)

func TestParseTXTRecord(t *testing.T) {
	tests := []struct {
		name    string
		record  string
		wantErr bool
		keyType string
	}{
		{
			name:    "valid ed25519 record",
			record:  "schema=v1; v=pubkey; k=ed25519; p=YWJjZGVmZ2hpamtsbW5vcHFyc3R1dnd4eXoxMjM0NTY=",
			wantErr: false,
			keyType: "ed25519",
		},
		{
			name:    "valid ecdsa record",
			record:  "schema=v1; v=pubkey; k=ecdsa-p256; p=YWJjZGVmZ2hpamtsbW5vcHFyc3R1dnd4eXoxMjM0NTY=",
			wantErr: false,
			keyType: "ecdsa-p256",
		},
		{
			name:    "missing schema",
			record:  "v=pubkey; k=ed25519; p=YWJjZGVmZ2hpamtsbW5vcHFyc3R1dnd4eXoxMjM0NTY=",
			wantErr: true,
		},
		{
			name:    "wrong schema version",
			record:  "schema=v2; v=pubkey; k=ed25519; p=YWJjZGVmZ2hpamtsbW5vcHFyc3R1dnd4eXoxMjM0NTY=",
			wantErr: true,
		},
		{
			name:    "missing value type",
			record:  "schema=v1; k=ed25519; p=YWJjZGVmZ2hpamtsbW5vcHFyc3R1dnd4eXoxMjM0NTY=",
			wantErr: true,
		},
		{
			name:    "wrong value type",
			record:  "schema=v1; v=unknown; k=ed25519; p=YWJjZGVmZ2hpamtsbW5vcHFyc3R1dnd4eXoxMjM0NTY=",
			wantErr: true,
		},
		{
			name:    "missing key type",
			record:  "schema=v1; v=pubkey; p=YWJjZGVmZ2hpamtsbW5vcHFyc3R1dnd4eXoxMjM0NTY=",
			wantErr: true,
		},
		{
			name:    "unsupported key type",
			record:  "schema=v1; v=pubkey; k=dsa; p=YWJjZGVmZ2hpamtsbW5vcHFyc3R1dnd4eXoxMjM0NTY=",
			wantErr: true,
		},
		{
			name:    "missing public key",
			record:  "schema=v1; v=pubkey; k=ed25519",
			wantErr: true,
		},
		{
			name:    "invalid base64",
			record:  "schema=v1; v=pubkey; k=ed25519; p=!!!invalid!!!",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, err := ParseTXTRecord(tt.record)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseTXTRecord(%q) expected error, got nil", tt.record)
				}

				return
			}

			if err != nil {
				t.Fatalf("ParseTXTRecord(%q) unexpected error: %v", tt.record, err)
			}

			if key.Type != tt.keyType {
				t.Errorf("ParseTXTRecord(%q).Type = %q, want %q", tt.record, key.Type, tt.keyType)
			}

			if len(key.Key) == 0 {
				t.Errorf("ParseTXTRecord(%q).Key is empty", tt.record)
			}
		})
	}
}
