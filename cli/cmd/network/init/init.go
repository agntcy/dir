// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package init

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"

	"github.com/agntcy/dir/cli/presenter"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/spf13/cobra"
)

var Command = &cobra.Command{
	Use:   "init",
	Short: "Generates the peer id from a newly generated private key, enabling connection to the DHT network",
	Long: `
This command generates a peer id from a newly generated private key. From this key
a peer id will be generated that is needed for the host to connect to the network.

Usage examples:

1. Generate peer id from a newly generated private key and save the key to the default location (~/.agntcy/dir/generated.key):

	dirctl network init

2. Generate peer id from a newly generated private key and save the key to a file:

	dirctl network init --output /path/to/private/key.pem

`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		return runCommand(cmd)
	},
}

func runCommand(cmd *cobra.Command) error {
	publicKey, privateKey, err := GenerateED25519OpenSSLKey()
	if err != nil {
		return fmt.Errorf("failed to generate ED25519 key pair: %w", err)
	}

	var filePath string
	if opts.Output != "" {
		filePath = opts.Output
	} else {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("error getting home directory: %w", err)
		}

		filePath = filepath.Join(homeDir, ".agntcy/dir/generated.key")
	}

	err = os.MkdirAll(filepath.Dir(filePath), 0o0755) //nolint:mnd
	if err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	err = os.WriteFile(filePath, privateKey, 0o600) //nolint:mnd
	if err != nil {
		return fmt.Errorf("failed to write private key to file: %w", err)
	}

	pubKey, err := crypto.UnmarshalEd25519PublicKey(publicKey)
	if err != nil {
		return fmt.Errorf("failed to unmarshal public key: %w", err)
	}

	ID, err := peer.IDFromPublicKey(pubKey)
	if err != nil {
		return fmt.Errorf("failed to generate peer ID from public key: %w", err)
	}

	presenter.Print(cmd, ID)

	return nil
}

func GenerateED25519OpenSSLKey() (ed25519.PublicKey, []byte, error) {
	// Generate ED25519 key pair
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate ED25519 key pair: %w", err)
	}

	// Marshal the private key to PKCS#8 format
	privBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal private key: %w", err)
	}

	// Create PEM block
	pemBlock := &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: privBytes,
	}

	return publicKey, pem.EncodeToMemory(pemBlock), nil
}
