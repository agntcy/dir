// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package types

import (
	ds "github.com/dep2p/libp2p/datastore"
)

type Database interface {
	Agent() ds.Datastore
}
