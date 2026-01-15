// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package wellknown

import (
	"testing"

	"github.com/lestrrat-go/jwx/v2/jwk"
)

func TestConvertJWKToPublicKey(t *testing.T) {
	tests := []struct {
		name    string
		jwkJSON string
		wantID  string
		wantErr bool
	}{
		{
			name: "EC P-256 key",
			jwkJSON: `{
				"kty": "EC",
				"crv": "P-256",
				"x": "MKBCTNIcKUSDii11ySs3526iDZ8AiTo7Tu6KPAqv7D4",
				"y": "4Etl6SRW2YiLUrN5vfvVHuhp7x8PxltmWWlbbM4IFyM",
				"kid": "ec-key-1"
			}`,
			wantID:  "ec-key-1",
			wantErr: false,
		},
		{
			name: "RSA key",
			jwkJSON: `{
				"kty": "RSA",
				"n": "0vx7agoebGcQSuuPiLJXZptN9nndrQmbXEps2aiAFbWhM78LhWx4cbbfAAtVT86zwu1RK7aPFFxuhDR1L6tSoc_BJECPebWKRXjBZCiFV4n3oknjhMstn64tZ_2W-5JsGY4Hc5n9yBXArwl93lqt7_RN5w6Cf0h4QyQ5v-65YGjQR0_FDW2QvzqY368QQMicAtaSqzs8KJZgnYb9c7d0zgdAZHzu6qMQvRL5hajrn1n91CbOpbISD08qNLyrdkt-bFTWhAI4vMQFh6WeZu0fM4lFd2NcRwr3XPksINHaQ-G_xBniIqbw0Ls1jF44-csFCur-kEgU8awapJzKnqDKgw",
				"e": "AQAB",
				"kid": "rsa-key-1"
			}`,
			wantID:  "rsa-key-1",
			wantErr: false,
		},
		{
			name: "Ed25519 key",
			jwkJSON: `{
				"kty": "OKP",
				"crv": "Ed25519",
				"x": "11qYAYKxCrfVS_7TyWQHOg7hcvPapiMlrwIaaPcHURo",
				"kid": "ed25519-key-1"
			}`,
			wantID:  "ed25519-key-1",
			wantErr: false,
		},
		{
			name: "key without kid",
			jwkJSON: `{
				"kty": "EC",
				"crv": "P-256",
				"x": "MKBCTNIcKUSDii11ySs3526iDZ8AiTo7Tu6KPAqv7D4",
				"y": "4Etl6SRW2YiLUrN5vfvVHuhp7x8PxltmWWlbbM4IFyM"
			}`,
			wantID:  "",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, err := jwk.ParseKey([]byte(tt.jwkJSON))
			if err != nil {
				t.Fatalf("Failed to parse JWK: %v", err)
			}

			publicKey, err := ConvertJWKToPublicKey(key)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got nil")
				}

				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if publicKey.ID != tt.wantID {
				t.Errorf("ID = %q, want %q", publicKey.ID, tt.wantID)
			}

			if len(publicKey.Key) == 0 {
				t.Error("Key bytes should not be empty")
			}

			if publicKey.Type == "" {
				t.Error("Type should not be empty")
			}
		})
	}
}

func TestConvertJWKToPublicKey_NilKey(t *testing.T) {
	_, err := ConvertJWKToPublicKey(nil)
	if err == nil {
		t.Error("Expected error for nil key")
	}
}
