// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package behavioral

import (
	"github.com/agntcy/dir/importer/scanner"
	scannerconfig "github.com/agntcy/dir/importer/scanner/config"
)

// Register the behavioral scanner with the factory on package init.
func init() {
	scanner.Register("behavioral", func(cfg scannerconfig.Config) scanner.Scanner { return New(cfg) })
}
