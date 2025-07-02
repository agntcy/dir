// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package database

import (
	"fmt"

	"github.com/agntcy/dir/server/database/sqlite"
	"github.com/agntcy/dir/server/types"
)

type DB string

const (
	SQLite DB = "sqlite"
)

func New(opts types.APIOptions) (types.DatabaseAPI, error) {
	switch db := DB(opts.Config().Database.DBType); db {
	case SQLite:
		sqliteDB, err := sqlite.New(opts.Config().Database.SQLite.DBPath)
		if err != nil {
			return nil, fmt.Errorf("failed to create SQLite database: %w", err)
		}

		return sqliteDB, nil
	default:
		return nil, fmt.Errorf("unsupported database=%s", db)
	}
}
