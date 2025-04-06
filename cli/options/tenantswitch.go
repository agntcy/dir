// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package options

type TenantSwitchOptions struct {
	*HubOptions
	*TenantOptions
}

func NewTenantSwitchOptions(hubOpts *HubOptions, tenantOptions *TenantOptions) *TenantSwitchOptions {
	return &TenantSwitchOptions{
		HubOptions:    hubOpts,
		TenantOptions: tenantOptions,
	}
}
