// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package cosign

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	_ "crypto/sha512" // if user chooses SHA2-384 or SHA2-512 for hash
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/sigstore/cosign/v2/pkg/cosign"
	"github.com/sigstore/cosign/v2/pkg/cosign/env"
	protocommon "github.com/sigstore/protobuf-specs/gen/pb-go/common/v1"
	"github.com/sigstore/sigstore/pkg/cryptoutils"
)

type KeypairOptions struct {
	Hint []byte
}

type Keypair struct {
	options       *KeypairOptions
	privateKey    crypto.Signer
	hashAlgorithm protocommon.HashAlgorithm
}

func LoadKeypair(privateKeyBytes []byte, pw []byte) (*Keypair, error) {
	if len(privateKeyBytes) == 0 {
		return nil, errors.New("private key bytes cannot be empty")
	}

	privateKey, err := cryptoutils.UnmarshalPEMToPrivateKey(
		privateKeyBytes,
		cryptoutils.StaticPasswordFunc(pw),
	)
	if err != nil {
		return nil, fmt.Errorf("unmarshal PEM to private key: %w", err)
	}

	// Get public key from the private key
	v, err := cosign.LoadPrivateKey(privateKeyBytes, pw)
	if err != nil {
		return nil, fmt.Errorf("failed to load private key: %w", err)
	}

	pubKey, err := v.PublicKey()
	if err != nil {
		return nil, fmt.Errorf("failed to get public key: %w", err)
	}

	// Derive the hint from the public key
	pubKeyBytes, err := cryptoutils.MarshalPublicKeyToDER(pubKey)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal public key: %w", err)
	}

	opts := &KeypairOptions{
		Hint: GenerateHintFromPublicKey(pubKeyBytes),
	}

	signer, ok := privateKey.(crypto.Signer)
	if !ok {
		return nil, errors.New("private key does not implement crypto.Signer")
	}

	return &Keypair{
		options:       opts,
		privateKey:    signer,
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
	switch pubKey := e.privateKey.Public().(type) {
	case *rsa.PublicKey:
		return "RSA"
	case *ecdsa.PublicKey:
		switch pubKey.Curve.Params().Name {
		case "P-256":
			return "ECDSA-P256"
		case "P-384":
			return "ECDSA-P384"
		case "P-521":
			return "ECDSA-P521"
		default:
			return "ECDSA"
		}
	case ed25519.PublicKey:
		return "Ed25519"
	default:
		return "Unknown"
	}
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

func ReadPrivateKeyPassword() func() ([]byte, error) {
	pw, ok := env.LookupEnv(env.VariablePassword)

	switch {
	case ok:
		return func() ([]byte, error) {
			return []byte(pw), nil
		}
	case cosign.IsTerminal():
		return func() ([]byte, error) {
			return cosign.GetPassFromTerm(true)
		}
	// Handle piped in passwords.
	default:
		return func() ([]byte, error) {
			return io.ReadAll(os.Stdin)
		}
	}
}
