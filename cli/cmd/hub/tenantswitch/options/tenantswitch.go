// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package options

import (
	hubOptions "github.com/agntcy/dir/cli/cmd/hub/options"
	commonOptions "github.com/agntcy/dir/cli/cmd/options"
	"github.com/spf13/cobra"
)

type TenantSwitchOptions struct {
	*hubOptions.HubOptions
	*commonOptions.TenantOptions
}

func NewTenantSwitchOptions(hubOpts *hubOptions.HubOptions, cmd *cobra.Command) *TenantSwitchOptions {
	return &TenantSwitchOptions{
		HubOptions:    hubOpts,
		TenantOptions: commonOptions.NewTenantOptions(hubOpts.BaseOption, cmd),
	}
}
