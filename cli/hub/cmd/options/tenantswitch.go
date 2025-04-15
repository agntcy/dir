// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package options

import (
	"github.com/spf13/cobra"
)

type TenantSwitchOptions struct {
	*HubOptions
	*TenantOption
}

func NewTenantSwitchOptions(hubOpts *HubOptions, cmd *cobra.Command) *TenantSwitchOptions {
	return &TenantSwitchOptions{
		HubOptions:   hubOpts,
		TenantOption: NewTenantOption(hubOpts, cmd),
	}
}
