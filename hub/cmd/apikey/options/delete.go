// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package options

import (
	"github.com/agntcy/dir/hub/cmd/options"
)

type APIKeyDeleteOptions struct {
	*options.HubOptions
}

func NewAPIKeyDeleteOptions(hubOptions *options.HubOptions) *APIKeyDeleteOptions {
	return &APIKeyDeleteOptions{
		HubOptions: hubOptions,
	}
}
