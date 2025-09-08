// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package options

import (
	"github.com/agntcy/dir/hub/cmd/options"
	"github.com/spf13/cobra"
)

type APIKeyDeleteOptions struct {
	*options.HubOptions

	JSONOutput bool
}

func NewAPIKeyDeleteOptions(hubOptions *options.HubOptions, cmd *cobra.Command) *APIKeyDeleteOptions {
	opt := &APIKeyDeleteOptions{
		HubOptions: hubOptions,
	}

	opt.AddRegisterFn(func() error {
		cmd.Flags().BoolVarP(&opt.JSONOutput, "json", "j", false, "Output in JSON format")

		return nil
	})

	return opt
}
