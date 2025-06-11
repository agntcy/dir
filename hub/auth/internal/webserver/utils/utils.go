// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package utils provides cryptographic and encoding utilities for PKCE and OAuth flows
// used by the internal webserver in the Agent Hub CLI and related applications.
package utils

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"math/big"
	"strings"
)

const (
	letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

	nonceLength             = 32
	challengeVerifierLength = 43
)

// RandStringBytes generates a random byte slice of the given length using a secure random source.
func RandStringBytes(n int) []byte {
	b := make([]byte, n)
	for i := range b {
		// Use cryptographically secure random number generation
		// to select a random byte from letterBytes
		index, err := rand.Int(rand.Reader, big.NewInt(int64(len(letterBytes))))
		if err != nil {
			panic(err)
		}

		b[i] = letterBytes[index.Int64()]
	}

	return b
}

// GenerateChallenge creates a PKCE code verifier and its corresponding S256 challenge.
// Returns the verifier and the base64-url-encoded challenge string.
func GenerateChallenge() (string, string) {
	verifier := RandStringBytes(challengeVerifierLength)

	hash := sha256.Sum256(verifier)
	challenge := base64.URLEncoding.EncodeToString(hash[:])
	challenge = strings.TrimRight(challenge, "=")

	return string(verifier), challenge
}

// GenerateNonce generates a random base64-url-encoded nonce string for use in OAuth flows.
func GenerateNonce() (string, error) {
	nonceBytes := make([]byte, nonceLength)

	_, err := rand.Read(nonceBytes)
	if err != nil {
		return "", errors.New("could not generate nonce")
	}

	return base64.URLEncoding.EncodeToString(nonceBytes), nil
}
