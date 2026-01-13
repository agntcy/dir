// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package verification

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
)

// ParseDNSTXTRecord parses an OASF DNS TXT record.
// Format: "v=oasf1; k=ed25519; p=<base64-encoded-public-key>".
//
//nolint:mnd
func ParseDNSTXTRecord(record string) (*PublicKey, error) {
	// Parse key-value pairs
	parts := strings.Split(record, ";")
	params := make(map[string]string)

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}

		key := strings.TrimSpace(kv[0])
		value := strings.TrimSpace(kv[1])
		params[key] = value
	}

	// Validate version
	version, ok := params["v"]
	if !ok || version != "oasf1" {
		return nil, fmt.Errorf("invalid or missing version: expected 'oasf1', got '%s'", version)
	}

	// Get key type
	keyType, ok := params["k"]
	if !ok {
		return nil, errors.New("missing key type parameter 'k'")
	}

	// Validate key type
	if !isValidKeyType(keyType) {
		return nil, fmt.Errorf("unsupported key type: %s", keyType)
	}

	// Get public key
	keyBase64, ok := params["p"]
	if !ok {
		return nil, errors.New("missing public key parameter 'p'")
	}

	// Decode base64
	keyBytes, err := base64.StdEncoding.DecodeString(keyBase64)
	if err != nil {
		// Try URL-safe base64
		keyBytes, err = base64.URLEncoding.DecodeString(keyBase64)
		if err != nil {
			// Try raw base64 (no padding)
			keyBytes, err = base64.RawStdEncoding.DecodeString(keyBase64)
			if err != nil {
				return nil, fmt.Errorf("invalid base64 public key: %w", err)
			}
		}
	}

	return &PublicKey{
		Type:      keyType,
		Key:       keyBytes,
		KeyBase64: keyBase64,
	}, nil
}

// ParseWellKnownKey converts a WellKnownKey to a PublicKey.
func ParseWellKnownKey(wk WellKnownKey) (*PublicKey, error) {
	// Validate key type
	if !isValidKeyType(wk.Type) {
		return nil, fmt.Errorf("unsupported key type: %s", wk.Type)
	}

	// Decode base64
	keyBytes, err := base64.StdEncoding.DecodeString(wk.PublicKey)
	if err != nil {
		// Try URL-safe base64
		keyBytes, err = base64.URLEncoding.DecodeString(wk.PublicKey)
		if err != nil {
			// Try raw base64 (no padding)
			keyBytes, err = base64.RawStdEncoding.DecodeString(wk.PublicKey)
			if err != nil {
				return nil, fmt.Errorf("invalid base64 public key: %w", err)
			}
		}
	}

	return &PublicKey{
		ID:        wk.ID,
		Type:      wk.Type,
		Key:       keyBytes,
		KeyBase64: wk.PublicKey,
	}, nil
}

// isValidKeyType checks if the key type is supported.
func isValidKeyType(keyType string) bool {
	switch strings.ToLower(keyType) {
	case "ed25519", "ecdsa-p256", "ecdsa-p384", "rsa":
		return true
	default:
		return false
	}
}

// MatchKey checks if the given signing key matches any of the domain's published keys.
// Returns the matched key and true if found, nil and false otherwise.
func MatchKey(signingKey []byte, domainKeys []PublicKey) (*PublicKey, bool) {
	for i := range domainKeys {
		if bytes.Equal(signingKey, domainKeys[i].Key) {
			return &domainKeys[i], true
		}
	}

	return nil, false
}
