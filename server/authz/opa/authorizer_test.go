// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package opa

import (
	"fmt"
	"testing"

	"github.com/agntcy/dir/server/authz/opa/config"
)

func TestNew(t *testing.T) {
	ctx := t.Context()
	cfg := config.Config{
		PolicyDirPath: "./testdata/policies",
	}

	authz, err := New(ctx, cfg)
	if err != nil {
		t.Fatalf("failed to create OPA authorizer: %v", err)
	}

	if authz == nil {
		t.Fatal("expected non-nil authorizer")
	}
}

func TestAuthorize(t *testing.T) {
	ctx := t.Context()
	cfg := config.Config{
		PolicyDirPath: "./testdata/policies",
	}

	authz, err := New(ctx, cfg)
	if err != nil {
		t.Fatalf("failed to create OPA authorizer: %v", err)
	}

	tests := []struct {
		userID  string
		request string
		allowed bool
	}{
		{"admin", "GET", true},
		{"admin", "POST", true},
		{"client", "GET", true},
		{"client", "POST", false},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s_%s", tt.userID, tt.request), func(t *testing.T) {
			if got := authz.Authorize(ctx, tt.userID, tt.request); got != tt.allowed {
				t.Errorf("Authorize() = %v, want %v", got, tt.allowed)
			}
		})
	}
}
