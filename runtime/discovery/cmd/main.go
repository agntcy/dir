// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/agntcy/dir/runtime/discovery"
	"github.com/agntcy/dir/runtime/discovery/config"
	"github.com/agntcy/dir/runtime/utils"
)

var logger = utils.NewLogger("runtime", "discovery")

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		logger.Error("failed to load configuration", "error", err)

		return
	}

	logger.Info("============================================================")
	logger.Info("Discovery Service")
	logger.Info("configuration loaded",
		"runtime", cfg.Runtime.Type,
		"storage", cfg.Store.Type,
		"workers", cfg.Workers,
	)
	logger.Info("============================================================")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Run the discovery service
	if err := discovery.Run(ctx,
		discovery.WithConfig(cfg),
		discovery.WithLogger(logger.Logger),
	); err != nil {
		logger.Error("discovery service failed", "error", err)
	}
}
