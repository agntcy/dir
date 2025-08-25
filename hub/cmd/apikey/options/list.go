// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package options

import (
	"github.com/agntcy/dir/hub/cmd/options"
	"github.com/spf13/cobra"
)

type ApiKeyListOptions struct {
	*options.HubOptions

	OrganizationId   string
	OrganizationName string
}

func NewApiKeyListOptions(hubOpts *options.HubOptions, cmd *cobra.Command) *ApiKeyListOptions {
	opt := &ApiKeyListOptions{
		HubOptions: hubOpts,
	}

	opt.AddRegisterFn(func() error {
		cmd.Flags().StringVarP(&opt.OrganizationId, "org-id", "o", "", "Organization ID")
		cmd.Flags().StringVarP(&opt.OrganizationName, "org-name", "n", "", "Organization Name")

		return nil
	})

	return opt
}
