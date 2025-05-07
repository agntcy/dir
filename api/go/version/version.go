// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package version

import "fmt"

// overridden using ldflags.
var (
	Version    string
	CommitHash string
)

func String() string {
	return fmt.Sprintf("%s (%s)", Version, CommitHash)
}
