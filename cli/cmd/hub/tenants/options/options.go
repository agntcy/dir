// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package options

import hubOptions "github.com/agntcy/dir/cli/cmd/hub/options"

type ListTenantsOptions struct {
	*hubOptions.HubOptions
}

func NewListTenantsOptions(hubOpts *hubOptions.HubOptions) *ListTenantsOptions {
	return &ListTenantsOptions{
		HubOptions: hubOpts,
	}
}
