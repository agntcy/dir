// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package gorm

import (
	"github.com/agntcy/dir/server/config"
	"github.com/agntcy/dir/server/database/types"
	ds "github.com/dep2p/libp2p/datastore"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type gormDB struct {
	db *gorm.DB
}

func (g *gormDB) Agent() ds.Datastore {
	return NewAgentTable(g.db)
}

func NewGorm(cfg *config.Config) (types.Database, error) {
	db, err := gorm.Open(sqlite.Open(cfg.DatabaseDSN), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	if err = db.AutoMigrate(&types.Agent{}); err != nil {
		return nil, err
	}

	return &gormDB{db: db}, nil
}
