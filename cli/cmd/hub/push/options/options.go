// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package options

import (
	hubOptions "github.com/agntcy/dir/cli/cmd/hub/options"
	pushOptions "github.com/agntcy/dir/cli/cmd/push/options"
)

type HubPushOptions struct {
	*hubOptions.HubOptions
	*pushOptions.PushOptions
}

func NewHubPushOptions(hubOptions *hubOptions.HubOptions, pushOptions *pushOptions.PushOptions) *HubPushOptions {
	return &HubPushOptions{
		HubOptions:  hubOptions,
		PushOptions: pushOptions,
	}
}
