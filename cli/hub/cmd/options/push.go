// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package options

import "github.com/spf13/cobra"

type HubPushOptions struct {
	*HubOptions

	FromStdIn bool
}

func NewHubPushOptions(hubOptions *HubOptions, cmd *cobra.Command) *HubPushOptions {
	opts := &HubPushOptions{
		HubOptions: hubOptions,
	}

	opts.AddRegisterFn(func() error {
		cmd.Flags().BoolVar(&opts.FromStdIn, "stdin", false, "Read from stdin")

		return nil
	})

	return opts
}
