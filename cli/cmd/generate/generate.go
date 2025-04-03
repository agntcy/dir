// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package generate

import (
	"crypto/ed25519"
	"fmt"
	"os"

	"github.com/agntcy/dir/cli/presenter"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
)

var Command = &cobra.Command{
	Use:   "generate",
	Short: "Generates the peer id from a private key, enabling connection to the DHT network",
	Long: `
This command requires a private key stored on the host filesystem. From this key
a peer id will be generated that is needed for the host to connect to the network.

Usage examples:

1. Generate peer id from secret key:

	dirctl generate --private-key-file-path <path-to-key>

`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		return runCommand(cmd)
	},
}

func runCommand(cmd *cobra.Command) error {
	// Read the SSH key file
	keyBytes, err := os.ReadFile(opts.PrivateKeyFilePath)
	if err != nil {
		return fmt.Errorf("failed to read key file: %w", err)
	}

	// Parse the private key
	key, err := ssh.ParseRawPrivateKey(keyBytes)
	if err != nil {
		return fmt.Errorf("failed to parse private key: %w", err)
	}

	// Try to convert to ED25519 private key
	ed25519Key, ok := key.(ed25519.PrivateKey)
	if !ok {
		return fmt.Errorf("key is not an ED25519 private key")
	}

	generatedKey, err := crypto.UnmarshalEd25519PrivateKey(ed25519Key)
	if err != nil {
		return fmt.Errorf("failed to unmarshal identity key: %w", err)
	}

	ID, err := peer.IDFromPublicKey(generatedKey.GetPublic())
	if err != nil {
		return fmt.Errorf("failed to generate peer ID from public key: %w", err)
	}

	presenter.Print(cmd, ID)

	return nil
}
