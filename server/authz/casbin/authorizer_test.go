package casbin

import (
	"context"
	"testing"
)

func TestAuthorizer(t *testing.T) {
	modelPath := "./testdata/model.conf"
	policyDir := "./testdata/policies/dir_com_policy.csv"
	authz, err := NewFromFiles(modelPath, policyDir)
	if err != nil {
		t.Fatalf("failed to create Casbin authorizer: %v", err)
	}

	tests := []struct {
		userID      string
		apiMethod   string
		trustDomain string
		allow       bool
	}{
		// dir.com: all users, all ops allowed
		{"spiffe://example.org/user/abc", "pull", "dir.com", true},
		{"spiffe://example.org/user/abc", "push", "dir.com", false},
	}

	for _, tt := range tests {
		allowed, err := authz.Authorize(context.Background(), tt.userID, tt.apiMethod, tt.trustDomain)
		if err != nil {
			t.Errorf("Authorize() error: %v", err)
		}
		if allowed != tt.allow {
			t.Errorf("Authorize(%q, %q, %q) = %v, want %v", tt.userID, tt.apiMethod, tt.trustDomain, allowed, tt.allow)
		}
	}
}
