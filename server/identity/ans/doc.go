// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package ans verifies ans:// identity claims through the Agent Name Service:
// the claim's signing key must belong to an identity certificate that the
// agent's transparency log attests for the agent the subject names.
package ans

import "github.com/agntcy/dir/utils/logging"

var logger = logging.Logger("identity/ans")
