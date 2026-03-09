// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package remote

import (
	"github.com/agntcy/dir/importer/scanner"
	scannerconfig "github.com/agntcy/dir/importer/scanner/config"
)

func init() {
	scanner.Register("remote", func(cfg scannerconfig.Config) scanner.Scanner { return New(cfg) })
}
