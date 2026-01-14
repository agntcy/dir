// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package wellknown

import (
	"encoding/base64"
	"fmt"

	"github.com/agntcy/dir/server/naming"
)

// ParseKey converts a WellKnownKey to a PublicKey.
func ParseKey(wk naming.WellKnownKey) (*naming.PublicKey, error) {
	// Validate key type
	if !naming.IsValidKeyType(wk.Type) {
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

	return &naming.PublicKey{
		ID:        wk.ID,
		Type:      wk.Type,
		Key:       keyBytes,
		KeyBase64: wk.PublicKey,
	}, nil
}
