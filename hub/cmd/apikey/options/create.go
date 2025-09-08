// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package options

import (
	"github.com/agntcy/dir/hub/cmd/options"
	"github.com/spf13/cobra"
)

type APIKeyCreateOptions struct {
	*options.HubOptions

	Role             string
	OrganizationID   string
	OrganizationName string
	JSONOutput       bool
}

func NewAPIKeyCreateOptions(hubOpts *options.HubOptions, cmd *cobra.Command) *APIKeyCreateOptions {
	opt := &APIKeyCreateOptions{
		HubOptions: hubOpts,
	}

	opt.AddRegisterFn(func() error {
		cmd.Flags().StringVarP(&opt.Role, "role", "r", "", "Role name. One of ['ROLE_ORG_ADMIN', 'ROLE_ADMIN', 'ROLE_EDITOR', 'ROLE_VIEWER']")
		cmd.Flags().StringVarP(&opt.OrganizationID, "org-id", "o", "", "Organization ID")
		cmd.Flags().StringVarP(&opt.OrganizationName, "org-name", "n", "", "Organization Name")
		cmd.Flags().BoolVarP(&opt.JSONOutput, "json", "j", false, "Output in JSON format")

		return nil
	})

	return opt
}
