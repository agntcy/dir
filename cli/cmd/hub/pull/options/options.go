// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package options

import hubOptions "github.com/agntcy/dir/cli/cmd/hub/options"

type HubPullOptions struct {
	*hubOptions.HubOptions
}

func NewHubPullOptions(hubOptions *hubOptions.HubOptions) *HubPullOptions {
	return &HubPullOptions{
		HubOptions: hubOptions,
	}
}
