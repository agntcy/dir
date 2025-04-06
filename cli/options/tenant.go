// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package options

import "github.com/spf13/cobra"

const tenantFlag = "tenant"

type TenantOptions struct {
	*BaseOption

	Tenant string
}

func NewTenantOptions(baseOption *BaseOption, cmd *cobra.Command) *TenantOptions {
	opts := &TenantOptions{
		BaseOption: baseOption,
	}

	opts.AddRegisterFns(func() error {
		flags := cmd.Flags()
		flags.StringVar(&opts.Tenant, tenantFlag, "", "Tenant name")

		return nil
	})

	return opts
}
