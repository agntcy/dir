// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package database

import (
	"testing"

	"github.com/agntcy/dir/server/config"
)

func TestDatabase(t *testing.T) {
	_, err := NewDatabase(&config.Config{
		DBDriver:    "gorm",
		DatabaseDSN: "/tmp/sqlite/dir.db",
	})

	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
}
