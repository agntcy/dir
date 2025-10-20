// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package mcp

import (
	"github.com/agntcy/dir/importer/config"
	"github.com/agntcy/dir/importer/types/factory"
)

// Register the MCP importer with the factory on package init.
func init() {
	factory.Register(config.RegistryTypeMCP, NewImporter)
}
