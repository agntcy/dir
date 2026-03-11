// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package behavioral

import (
	scannerconfig "github.com/agntcy/dir/importer/scanner/config"
	"github.com/agntcy/dir/importer/scanner/factory"
	"github.com/agntcy/dir/importer/scanner/types"
)

// Register the behavioral scanner with the factory on package init.
func init() {
	factory.Register("behavioral", func(cfg scannerconfig.Config) types.Scanner { return New(cfg) })
}
