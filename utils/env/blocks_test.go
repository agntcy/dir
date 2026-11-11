// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package env

import (
	"strings"
	"testing"
)

var testKeys = []string{
	"listen_address",
	"reconciler.policy_evaluation.enabled",
	"reconciler.policy.dir",
	"server.store.provider",
}

func TestCheckBlocks_RefusesAVariableOverABlock(t *testing.T) {
	for _, variable := range []string{
		"TESTBLK_RECONCILER",
		"TESTBLK_RECONCILER_POLICY_EVALUATION",
		"TESTBLK_RECONCILER_POLICY",
		"TESTBLK_SERVER",
		"TESTBLK_SERVER_STORE",
	} {
		for _, value := range []string{"", "x"} {
			t.Run(variable+"="+value, func(t *testing.T) {
				t.Setenv(variable, value)

				err := CheckBlocks("TESTBLK", testKeys)
				if err == nil {
					t.Fatalf("expected %s to be refused", variable)
				}

				if got := err.Error(); !strings.Contains(got, variable+" is set") {
					t.Fatalf("error does not name %s: %s", variable, got)
				}
			})
		}
	}
}

func TestCheckBlocks_AcceptsSettings(t *testing.T) {
	t.Setenv("TESTBLK_LISTEN_ADDRESS", "")
	t.Setenv("TESTBLK_RECONCILER_POLICY_EVALUATION_ENABLED", "true")
	t.Setenv("TESTBLK_SERVER_STORE_PROVIDER", "oci")

	if err := CheckBlocks("TESTBLK", testKeys); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCheckBlocks_IgnoresAnotherPrefix(t *testing.T) {
	t.Setenv("OTHERBLK_RECONCILER", "")

	if err := CheckBlocks("TESTBLK", testKeys); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCheckBlocks_LeafSharingABlocksVariableIsASetting(t *testing.T) {
	// server_store and server.store map to the same variable; the leaf wins.
	keys := []string{"server_store", "server.store.provider"}

	t.Setenv("TESTBLK_SERVER_STORE", "x")

	if err := CheckBlocks("TESTBLK", keys); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
