package utils

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"math/rand"
	"strings"
)

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func RandStringBytes(n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return b
}

func GenerateChallenge() (string, string) {
	verifier := RandStringBytes(43)

	hash := sha256.Sum256(verifier)
	challenge := base64.URLEncoding.EncodeToString(hash[:])
	challenge = strings.TrimRight(challenge, "=")

	return string(verifier), challenge
}

func GenerateNonce() (string, error) {
	nonceBytes := make([]byte, 32)
	_, err := rand.Read(nonceBytes)
	if err != nil {
		return "", fmt.Errorf("could not generate nonce")
	}

	return base64.URLEncoding.EncodeToString(nonceBytes), nil
}
