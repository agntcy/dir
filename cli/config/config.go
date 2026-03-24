// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"github.com/agntcy/dir/client"
)

// Client holds the merged CLI config (from config file, env, and flags).
// It is populated by cmd/options.go init and updated when flags are parsed.
var Client *client.Config = &client.DefaultConfig
