// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package options

import (
	"github.com/agntcy/dir/hub/cmd/options"
	"github.com/spf13/cobra"
)

type ApiKeyCreateOptions struct {
	*options.HubOptions

	Role             string
	OrganizationId   string
	OrganizationName string
}

func NewApiKeyCreateOptions(hubOpts *options.HubOptions, cmd *cobra.Command) *ApiKeyCreateOptions {
	opt := &ApiKeyCreateOptions{
		HubOptions: hubOpts,
	}

	opt.AddRegisterFn(func() error {
		cmd.Flags().StringVarP(&opt.Role, "role", "r", "", "Role name. One of ['ROLE_ORG_ADMIN', 'ROLE_ADMIN', 'ROLE_EDITOR', 'ROLE_VIEWER']")
		cmd.Flags().StringVarP(&opt.OrganizationId, "org-id", "o", "", "Organization ID")
		cmd.Flags().StringVarP(&opt.OrganizationName, "org-name", "n", "", "Organization Name")

		return nil
	})

	return opt
}
