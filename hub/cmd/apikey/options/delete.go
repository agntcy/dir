// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package options

import (
	"github.com/agntcy/dir/hub/cmd/options"
)

type ApiKeyDeleteOptions struct {
	*options.HubOptions
}

func NewApiKeyDeleteOptions(hubOptions *options.HubOptions) *ApiKeyDeleteOptions {
	return &ApiKeyDeleteOptions{
		HubOptions: hubOptions,
	}
}
