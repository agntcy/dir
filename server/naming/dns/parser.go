// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package dns

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"github.com/agntcy/dir/server/naming"
)

// ParseTXTRecord parses an OASF DNS TXT record.
// Format: "v=oasf1; k=ed25519; p=<base64-encoded-public-key>".
//
//nolint:mnd
func ParseTXTRecord(record string) (*naming.PublicKey, error) {
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
	if !naming.IsValidKeyType(keyType) {
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

	return &naming.PublicKey{
		Type:      keyType,
		Key:       keyBytes,
		KeyBase64: keyBase64,
	}, nil
}
