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
		BundlePath: "./testdata/policies",
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
		BundlePath: "./testdata/policies",
	}

	authz, err := New(ctx, cfg)
	if err != nil {
		t.Fatalf("failed to create OPA authorizer: %v", err)
	}

	tests := []struct {
		trustDomain string
		userID      string
		apiMethod   string
		allowed     bool
	}{
		{"dir.com", "spiffe://dir.com/admin", "PushRequest", true},
		{"dir.com", "spiffe://dir.com/admin", "LookupRequest", true},
		{"dir.com", "spiffe://dir.com/admin", "PullRequest", true},
		{"dir.com", "spiffe://dir.com/admin", "DeleteRequest", true},
		{"service.org", "spiffe://service.org/client", "PushRequest", false},
		{"service.org", "spiffe://service.org/client", "LookupRequest", true},
		{"service.org", "spiffe://service.org/client", "PullRequest", true},
		{"service.org", "spiffe://service.org/client", "DeleteRequest", false},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s_%s_%s", tt.trustDomain, tt.userID, tt.apiMethod), func(t *testing.T) {
			got, err := authz.Authorize(ctx, tt.trustDomain, tt.userID, tt.apiMethod)
			if err != nil {
				t.Errorf("Authorize() error = %v", err)
				return
			}
			if got != tt.allowed {
				t.Errorf("Authorize() = %v, want %v", got, tt.allowed)
			}
		})
	}
}
