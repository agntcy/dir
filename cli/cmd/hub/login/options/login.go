// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package options

import (
	hubOptions "github.com/agntcy/dir/cli/cmd/hub/options"
)

type LoginOptions struct {
	*hubOptions.HubOptions
}

func NewLoginOptions(hubOptions *hubOptions.HubOptions) *LoginOptions {
	return &LoginOptions{
		HubOptions: hubOptions,
	}
}
