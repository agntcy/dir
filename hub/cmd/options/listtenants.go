// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package options

type ListTenantsOptions struct {
	*HubOptions
}

func NewListTenantsOptions(hubOpts *HubOptions) *ListTenantsOptions {
	return &ListTenantsOptions{
		HubOptions: hubOpts,
	}
}
