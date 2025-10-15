// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package mcp

import (
	"github.com/agntcy/dir/importer/config"
	"github.com/agntcy/dir/importer/types"
)

// Register registers the MCP importer with the factory.
func Register(factory *types.Factory) {
	factory.Register(config.RegistryTypeMCP, NewImporter)
}
