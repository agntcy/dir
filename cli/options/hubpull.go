// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package options

type HubPullOptions struct {
	*HubOptions
}

func NewHubPullOptions(hubOptions *HubOptions) *HubPullOptions {
	return &HubPullOptions{
		HubOptions: hubOptions,
	}
}
