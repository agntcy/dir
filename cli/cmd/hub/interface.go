// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package hub

import "context"

type Hub interface {
	Run(ctx context.Context, args []string) error
}
