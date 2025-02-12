// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package search

import (
	"github.com/spf13/cobra"
)

var Command = &cobra.Command{
	Use:   "search",
	Short: "Search agents based on passed query",
	Long: `Usage example:

	dirctl search --query "agent that performs linkedin posts"

`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		return runCommand(cmd)
	},
}

func runCommand(cmd *cobra.Command) error {
	// TODO(paralta) Implement search command
	// 1. Connect to Registry/Search Client
	// 2. Search agents based on query

	return nil
}
