// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package env_test

import (
	"testing"

	"github.com/agntcy/dir/utils/env"
	"github.com/stretchr/testify/assert"
)

// The names viper derives, for the three prefixes in use and for both
// separators a key can carry.
func TestName(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		prefix, key, want string
	}{
		{"DIRECTORY_SERVER", "policy.refresh_interval", "DIRECTORY_SERVER_POLICY_REFRESH_INTERVAL"},
		{"DIRECTORY_DAEMON", "server.policy.dir", "DIRECTORY_DAEMON_SERVER_POLICY_DIR"},
		{"RECONCILER", "policy_evaluation.batch_size", "RECONCILER_POLICY_EVALUATION_BATCH_SIZE"},
		{"DIRECTORY_SERVER", "store.oci.auth-config", "DIRECTORY_SERVER_STORE_OCI_AUTH_CONFIG"},
	} {
		t.Run(tt.want, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, env.Name(tt.prefix, tt.key))
		})
	}
}
