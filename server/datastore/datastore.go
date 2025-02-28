// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package datastore

import (
	"github.com/agntcy/dir/server/config"
	"github.com/agntcy/dir/server/types"
	"github.com/ipfs/go-datastore"
)

// New is shortcut to creating specific datastore.
// For now, we use memory store.
//
// We should only create a proper datastore from options,
// as we do not implement this interface.
func New(_ *config.Config) (types.Datastore, error) {
	return datastore.NewMapDatastore(), nil
}
