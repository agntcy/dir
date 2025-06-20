// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	_ "crypto/sha512" // if user chooses SHA2-384 or SHA2-512 for hash
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"

	"github.com/secure-systems-lab/go-securesystemslib/encrypted"
	"github.com/sigstore/cosign/v2/pkg/cosign"
	protocommon "github.com/sigstore/protobuf-specs/gen/pb-go/common/v1"
	"github.com/sigstore/sigstore/pkg/cryptoutils"
)

type KeypairOptions struct {
	Hint []byte
}

type Keypair struct {
	options       *KeypairOptions
	privateKey    *ecdsa.PrivateKey
	hashAlgorithm protocommon.HashAlgorithm
}

func NewKeypair(privateKeyBytes []byte) (*Keypair, error) {
	if len(privateKeyBytes) == 0 {
		return nil, errors.New("private key bytes cannot be empty")
	}

	// Decode the PEM-encoded private key
	p, _ := pem.Decode(privateKeyBytes)
	if p == nil {
		return nil, errors.New("failed to decode PEM block containing private key")
	}

	if p.Type != cosign.CosignPrivateKeyPemType && p.Type != cosign.SigstorePrivateKeyPemType {
		return nil, fmt.Errorf("unsupported pem type: %s", p.Type)
	}

	// Decrypt the private key if it is encrypted and parse it
	x509Encoded, err := encrypted.Decrypt(p.Bytes, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt: %w", err)
	}

	privateKey, err := x509.ParsePKCS8PrivateKey(x509Encoded)
	if err != nil {
		return nil, fmt.Errorf("parsing private key: %w", err)
	}

	// Get public key from the private key
	v, err := cosign.LoadPrivateKey(privateKeyBytes, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to load private key: %w", err)
	}

	pubKey, err := v.PublicKey()
	if err != nil {
		return nil, fmt.Errorf("failed to get public key: %w", err)
	}

	// Derive the hint from the public key
	pubKeyBytes, err := x509.MarshalPKIXPublicKey(pubKey)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal public key: %w", err)
	}

	opts := &KeypairOptions{
		Hint: GenerateHintFromPublicKey(pubKeyBytes),
	}

	// Ensure the private key is of type *ecdsa.PrivateKey
	ecdsaPrivKey, ok := privateKey.(*ecdsa.PrivateKey)
	if !ok {
		return nil, errors.New("private key is not of type *ecdsa.PrivateKey")
	}

	return &Keypair{
		options:       opts,
		privateKey:    ecdsaPrivKey,
		hashAlgorithm: protocommon.HashAlgorithm_SHA2_256,
	}, nil
}

func (e *Keypair) GetHashAlgorithm() protocommon.HashAlgorithm {
	return e.hashAlgorithm
}

func (e *Keypair) GetHint() []byte {
	return e.options.Hint
}

func (e *Keypair) GetKeyAlgorithm() string {
	return "ECDSA"
}

func (e *Keypair) GetPublicKeyPem() (string, error) {
	pubKeyBytes, err := cryptoutils.MarshalPublicKeyToPEM(e.privateKey.Public())
	if err != nil {
		return "", fmt.Errorf("failed to marshal public key to PEM: %w", err)
	}

	return string(pubKeyBytes), nil
}

func (e *Keypair) SignData(_ context.Context, data []byte) ([]byte, []byte, error) {
	hasher := crypto.SHA256.New()
	hasher.Write(data)
	digest := hasher.Sum(nil)

	signature, err := e.privateKey.Sign(rand.Reader, digest, crypto.SHA256)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to sign data: %w", err)
	}

	return signature, digest, nil
}

func GenerateHintFromPublicKey(pubKey []byte) []byte {
	hashedBytes := sha256.Sum256(pubKey)

	return []byte(base64.StdEncoding.EncodeToString(hashedBytes[:]))
}
