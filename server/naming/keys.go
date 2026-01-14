// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package naming

import (
	"bytes"
	"strings"
)

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

// IsValidKeyType checks if the key type is supported.
func IsValidKeyType(keyType string) bool {
	switch strings.ToLower(keyType) {
	case "ed25519", "ecdsa-p256", "ecdsa-p384", "rsa":
		return true
	default:
		return false
	}
}
