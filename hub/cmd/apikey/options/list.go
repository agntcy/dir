// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package options

import (
	"github.com/agntcy/dir/hub/cmd/options"
	"github.com/spf13/cobra"
)

type APIKeyListOptions struct {
	*options.HubOptions

	OrganizationID   string
	OrganizationName string
}

func NewAPIKeyListOptions(hubOpts *options.HubOptions, cmd *cobra.Command) *APIKeyListOptions {
	opt := &APIKeyListOptions{
		HubOptions: hubOpts,
	}

	opt.AddRegisterFn(func() error {
		cmd.Flags().StringVarP(&opt.OrganizationID, "org-id", "o", "", "Organization ID")
		cmd.Flags().StringVarP(&opt.OrganizationName, "org-name", "n", "", "Organization Name")

		return nil
	})

	return opt
}
