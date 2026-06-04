// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package org

import "context"

// Resolver expands a manager alias into a flat list of aliases that includes
// the manager and all direct and indirect reports.
type Resolver interface {
	Expand(ctx context.Context, managerAlias string) ([]string, error)
}
