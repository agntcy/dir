// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package v1

import (
	"testing"

	ocidigest "github.com/opencontainers/go-digest"
)

func TestCalculateDigest(t *testing.T) {
	// Test cases
	tests := []struct {
		name       string
		data       []byte
		wantDigest string
		wantErr    bool
	}{
		{
			name:    "Empty data",
			data:    []byte{},
			wantErr: true,
		},
		{
			name:       "Hello World",
			data:       []byte("Hello, World!"),
			wantDigest: "sha256:dffd6021bb2bd5b0af676290809ec3a53191dd81c7f70a4b28688a362182986f",
		},
		{
			name:       "Random data",
			data:       []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05},
			wantDigest: "sha256:17e88db187afd62c16e5debf3e6527cd006bc012bc90b51a810cd80c2d511f43",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotDigest, err := CalculateDigest(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("CalculateDigest() error = %v, wantErr %v", err, tt.wantErr)

				return
			}

			if gotDigest.String() != tt.wantDigest {
				t.Errorf("CalculateDigest() = %v, want %v", gotDigest, tt.wantDigest)
			}
		})
	}
}

func TestConvertDigestToCID(t *testing.T) {
	// Test cases
	tests := []struct {
		name    string
		digest  string
		wantCID string
		wantErr bool
	}{
		{
			name:    "Empty digest",
			digest:  "",
			wantErr: true,
		},
		{
			name:    "Unsupported algorithm",
			digest:  "md5:d41d8cd98f00b204e9800998ecf8427e",
			wantErr: true,
		},
		{
			name:    "Invalid digest format",
			digest:  "d41d8cd98f00b204e9800998ecf8427e",
			wantErr: true,
		},
		{
			name:    "Valid SHA256 digest",
			digest:  "sha256:dffd6021bb2bd5b0af676290809ec3a53191dd81c7f70a4b28688a362182986f",
			wantCID: "baeareig77vqcdozl2wyk6z3cscaj5q5fggi53aoh64fewkdiri3cdauyn4",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotCID, err := ConvertDigestToCID(ocidigest.Digest(tt.digest))
			if (err != nil) != tt.wantErr {
				t.Errorf("ConvertDigestToCID() error = %v, wantErr %v", err, tt.wantErr)

				return
			}

			if gotCID != tt.wantCID {
				t.Errorf("ConvertDigestToCID() = %v, want %v", gotCID, tt.wantCID)
			}
		})
	}
}

func TestConvertCIDToDigest(t *testing.T) {
	// Test cases
	tests := []struct {
		name       string
		cid        string
		wantDigest string
		wantErr    bool
	}{
		{
			name:    "Empty CID",
			cid:     "",
			wantErr: true,
		},
		{
			name:    "Invalid CID format",
			cid:     "invalid-cid-string",
			wantErr: true,
		},
		{
			name:    "Unsupported hash type in CID",
			cid:     "bafkreigh2akiscaildc7t5j5x3t6l5g7x7y6z7x7y6z7x7y6z7x7y6z7x7y6z", // This is a CID with a different hash type
			wantErr: true,
		},
		{
			name:       "Valid CID",
			cid:        "baeareig77vqcdozl2wyk6z3cscaj5q5fggi53aoh64fewkdiri3cdauyn4",
			wantDigest: "sha256:dffd6021bb2bd5b0af676290809ec3a53191dd81c7f70a4b28688a362182986f",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotDigest, err := ConvertCIDToDigest(tt.cid)
			if (err != nil) != tt.wantErr {
				t.Errorf("ConvertCIDToDigest() error = %v, wantErr %v", err, tt.wantErr)

				return
			}

			if gotDigest.String() != tt.wantDigest {
				t.Errorf("ConvertCIDToDigest() = %v, want %v", gotDigest.String(), tt.wantDigest)
			}
		})
	}
}

func TestIsValidCID(t *testing.T) {
	// Test cases
	tests := []struct {
		name      string
		cid       string
		wantValid bool
	}{
		{
			name:      "Valid CID",
			cid:       "baeareig77vqcdozl2wyk6z3cscaj5q5fggi53aoh64fewkdiri3cdauyn4",
			wantValid: true,
		},
		{
			name:      "Invalid CID - wrong format",
			cid:       "invalid-cid-string",
			wantValid: false,
		},
		{
			name:      "Empty CID",
			cid:       "",
			wantValid: false,
		},
		{
			name:      "CID with invalid characters",
			cid:       "bafybeigdyrzt5tqz5f5u3j5x3t6l5g7x7y6z7x7y6z7x7y6z7x7y6z7x7y6z7x!",
			wantValid: false,
		},
		{
			name:      "CID with spaces",
			cid:       "bafybeigdyrzt 5tqz5f5u3j5x3t6l5g7x7y6z7x7y6z7x7y6z7x7y6z7x",
			wantValid: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotValid := IsValidCID(tt.cid); gotValid != tt.wantValid {
				t.Errorf("IsValidCID() = %v, want %v", gotValid, tt.wantValid)
			}
		})
	}
}
