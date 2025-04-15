// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package options

import "github.com/spf13/cobra"

type TenantOption struct {
	*HubOptions

	Tenant string
}

func NewTenantOption(hubOpts *HubOptions, cmd *cobra.Command) *TenantOption {
	opt := &TenantOption{
		HubOptions: hubOpts,
	}

	opt.AddRegisterFn(func() error {
		cmd.Flags().StringVarP(&opt.Tenant, "tenant", "t", "", "Tenant name")

		return nil
	})

	return opt
}
