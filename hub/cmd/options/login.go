// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package options

type LoginOptions struct {
	*HubOptions
}

func NewLoginOptions(hubOptions *HubOptions) *LoginOptions {
	return &LoginOptions{
		HubOptions: hubOptions,
	}
}
