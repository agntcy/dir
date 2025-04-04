// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package info

import (
	"crypto/ed25519"
	"errors"
	"fmt"
	"os"

	"github.com/agntcy/dir/cli/presenter"
	"github.com/agntcy/dir/utils/logging"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
)

var logger = logging.Logger("cli/network/info")

var Command = &cobra.Command{
	Use:   "info",
	Short: "Generates the peer id from a private key, enabling connection to the DHT network",
	Long: `
This command requires a private key stored on the host filesystem. From this key
a peer id will be generated that is needed for the host to connect to the network.

Usage examples:

1. Get peer id from a private key:

	dirctl network info <path_to_private_key>

`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return errors.New("expected exactly one argument")
		}

		if args[0] == "" {
			return errors.New("expected a non-empty argument")
		}

		return runCommand(cmd, args[0])
	},
}

func runCommand(cmd *cobra.Command, path string) error {
	logger.Info("Generating peer ID from key on the filesystem path", "path", path)

	// Read the SSH key file
	keyBytes, err := os.ReadFile(path)
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
		return errors.New("key is not an ED25519 private key")
	}

	// Generate a private key from bytes
	generatedKey, err := crypto.UnmarshalEd25519PrivateKey(ed25519Key)
	if err != nil {
		return fmt.Errorf("failed to unmarshal identity key: %w", err)
	}

	// Generate Peer ID from the public key
	ID, err := peer.IDFromPublicKey(generatedKey.GetPublic())
	if err != nil {
		return fmt.Errorf("failed to generate peer ID from public key: %w", err)
	}

	presenter.Print(cmd, ID)

	return nil
}
