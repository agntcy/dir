// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package version

import (
	"github.com/agntcy/dir/api/core/v1alpha1/version"
	"github.com/agntcy/dir/cli/presenter"
	"github.com/spf13/cobra"
)

var Command = &cobra.Command{
	Use:   "version",
	Short: "Print the version of the application",
	Run: func(cmd *cobra.Command, _ []string) {
		presenter.Print(cmd, "Application Version:", version.String())
	},
}
