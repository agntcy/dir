// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package routing

import (
	"context"

	coretypes "github.com/agntcy/dir/api/core/v1alpha1"
)

// TODO: implement what happens after publish event, ie. how we sync received data.
func (r *routing) sync(ctx context.Context, ref *coretypes.ObjectRef) error {
	panic("unimplemented")
}
