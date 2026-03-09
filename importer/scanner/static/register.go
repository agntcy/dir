// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package static

import (
	"github.com/agntcy/dir/importer/scanner"
	scannerconfig "github.com/agntcy/dir/importer/scanner/config"
)

func init() {
	scanner.Register("static", func(cfg scannerconfig.Config) scanner.Scanner { return New(cfg) })
}
