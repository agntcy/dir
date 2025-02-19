// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package database

import (
	"fmt"

	"github.com/agntcy/dir/server/config"
	"github.com/agntcy/dir/server/database/gorm"
	"github.com/agntcy/dir/server/database/types"
)

type Driver string

const (
	GORM = Driver("gorm")
)

func NewDatabase(cfg *config.Config) (types.Database, error) {
	switch driver := Driver(cfg.DBDriver); driver {
	case GORM:
		return gorm.NewGorm(cfg)
	default:
		return nil, fmt.Errorf("unsupported driver: %s", driver)
	}
}
