// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package initialize

import (
	"github.com/agntcy/dir/cli/cmd/initialize/repo"
	"github.com/spf13/cobra"
)

var Command = &cobra.Command{
	Use:   "init",
	Short: "CLI tool to initialize different components",
	Long: `This command initializes different components of the Directory system. Currently, it supports initializing
a new agent.json file.

`,
}

func init() {
	Command.AddCommand(
		repo.Command,
	)
}
