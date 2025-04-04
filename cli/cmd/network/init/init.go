// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package init

import (
	"crypto/rand"
	"fmt"

	"github.com/agntcy/dir/cli/presenter"
	"github.com/agntcy/dir/utils/logging"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/spf13/cobra"
)

var logger = logging.Logger("cli/network/init")

var Command = &cobra.Command{
	Use:   "init",
	Short: "Generates the peer id from a newly generated private key, enabling connection to the DHT network",
	Long: `
This command generates a peer id from a newly generated private key. From this key
a peer id will be generated that is needed for the host to connect to the network.

Usage examples:

1. Generate peer id from a newly generated private key:

	dirctl network init

`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		return runCommand(cmd)
	},
}

func runCommand(cmd *cobra.Command) error {
	logger.Info("Generating peer ID from a newly generated private key")

	generatedKey, _, err := crypto.GenerateKeyPairWithReader(
		crypto.Ed25519, // Select your key type. Ed25519 are nice short
		-1,             // Select key length when possible (i.e. RSA).
		rand.Reader,    // Always generate a random ID
	)
	if err != nil {
		return fmt.Errorf("failed to create identity key: %w", err)
	}

	// Generate Peer ID from the public key
	ID, err := peer.IDFromPublicKey(generatedKey.GetPublic())
	if err != nil {
		return fmt.Errorf("failed to generate peer ID from public key: %w", err)
	}

	presenter.Print(cmd, ID)

	return nil
}
