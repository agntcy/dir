// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package options

type HubPushOptions struct {
	*HubOptions
	*PushOptions
}

func NewHubPushOptions(hubOptions *HubOptions, pushOptions *PushOptions) *HubPushOptions {
	return &HubPushOptions{
		HubOptions:  hubOptions,
		PushOptions: pushOptions,
	}
}
