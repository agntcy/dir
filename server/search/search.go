// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package search

import (
	"fmt"

	"github.com/agntcy/dir/server/search/sqlite"
	"github.com/agntcy/dir/server/types"
)

type DB string

const (
	SQLite DB = "sqlite"
)

func New(opts types.APIOptions) (types.SearchAPI, error) {
	switch db := DB(opts.Config().Search.DBType); db {
	case SQLite:
		sqliteDB, err := sqlite.New(opts.Config().Search.SQLite.DBPath)
		if err != nil {
			return nil, fmt.Errorf("failed to create SQLite search database: %w", err)
		}

		return sqliteDB, nil
	default:
		return nil, fmt.Errorf("unsupported search database=%s", db)
	}
}
