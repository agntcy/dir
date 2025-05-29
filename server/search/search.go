// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package search

import (
	"fmt"

	"github.com/agntcy/dir/server/search/sqlite"
	searchtypes "github.com/agntcy/dir/server/search/types"
	"github.com/agntcy/dir/server/types"
)

type DB string

const (
	SQLite DB = "sqlite"
)

func New(opts types.APIOptions) (searchtypes.SearchAPI, error) {
	switch db := DB(opts.Config().Search.DB); db {
	case SQLite:
		sqliteDB, err := sqlite.New()
		if err != nil {
			return nil, fmt.Errorf("failed to create SQLite search database: %w", err)
		}

		return sqliteDB, nil
	default:
		return nil, fmt.Errorf("unsupported search database=%s", db)
	}
}
