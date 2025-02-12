// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/agntcy/dir/cli/cmd/build"
	"github.com/agntcy/dir/cli/cmd/pull"
	"github.com/agntcy/dir/cli/cmd/push"
	"github.com/agntcy/dir/cli/cmd/search"
	"github.com/agntcy/dir/cli/util"
	"github.com/agntcy/dir/registry/client"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "dirctl",
	Short: "CLI tool to interact with Directory",
	Long:  ``,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		c, err := client.New()
		if err != nil {
			return fmt.Errorf("failed to create registry client: %w", err)
		}

		ctx := util.SetRegistryClientForContext(cmd.Context(), c)
		cmd.SetContext(ctx)

		return nil
	},
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGHUP, syscall.SIGTERM)
	defer func() {
		cancel()
	}()

	if err := rootCmd.ExecuteContext(ctx); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// TODO: format commands output to avoid cleanup
	_, _ = fmt.Fprintf(rootCmd.OutOrStdout(), "\n")
}

func init() {
	// TODO register CLI flags

	// Register commands
	rootCmd.AddCommand(build.Command)
	rootCmd.AddCommand(pull.Command)
	rootCmd.AddCommand(push.Command)
	rootCmd.AddCommand(search.Command)
}
