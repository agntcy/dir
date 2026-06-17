// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// nolint
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"protocol_interop/consumer"
	"protocol_interop/publisher"

	clicmd "github.com/agntcy/dir/cli/cmd"
	adsclient "github.com/agntcy/dir/client"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// ================================
	// Setup
	// ================================

	// Start a local ADS daemon via CLI
	go func() {
		_ = startDaemon(ctx)
	}()

	time.Sleep(2 * time.Second) // wait for daemon to start

	// Create local client
	client, err := adsclient.New(ctx, adsclient.WithConfig(&adsclient.Config{
		ServerAddress: adsclient.DefaultServerAddress,
		AuthMode:      "none",
	}))
	if err != nil {
		fmt.Printf("Error creating ADS client: %v\n", err)

		return
	}
	defer client.Close()

	// ================================
	// Run the producer
	// ================================
	if err := publisher.Publish(ctx, client); err != nil {
		fmt.Printf("Error publishing ads: %v\n", err)

		return
	}

	// ================================
	// Run the consumer
	// ================================
	consumer.Handle(ctx, client, "Find some agents that can perform image segmentation tasks.")
}

func startDaemon(ctx context.Context) error {
	// Create a temporary directory for the ADS daemon data
	tmpDir, err := os.MkdirTemp("", "ads-daemon-*")
	if err != nil {
		return fmt.Errorf("failed to create temporary directory for ADS daemon: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Start the ADS daemon with the temporary data directory
	clicmd.RootCmd.SetArgs([]string{"daemon", "start", "--data-dir", tmpDir})

	err = clicmd.RootCmd.ExecuteContext(ctx)
	if err != nil && !errors.Is(err, context.Canceled) {
		return fmt.Errorf("failed to start ADS daemon: %w", err)
	}

	return nil
}
