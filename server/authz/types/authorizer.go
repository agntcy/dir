// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package types

import "context"

// Authorizer defines the interface for authorization.
// It checks if a user is allowed to perform a specific request.
type Authorizer interface {
	Authorize(ctx context.Context, trustDomain, userID, apiMethod string) bool
}
