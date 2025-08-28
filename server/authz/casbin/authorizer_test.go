package casbin

import (
	"context"
	"testing"

	routingv1 "github.com/agntcy/dir/api/routing/v1"
	storev1 "github.com/agntcy/dir/api/store/v1"
	"github.com/agntcy/dir/server/authz/config"
)

func TestAuthorizer(t *testing.T) {
	authz, err := New(config.Config{
		TrustDomain: "dir.com",
	})
	if err != nil {
		t.Fatalf("failed to create Casbin authorizer: %v", err)
	}

	tests := []struct {
		trustDomain string
		apiMethod   string
		allow       bool
	}{
		// dir.com: all ops allowed
		{"dir.com", storev1.StoreService_Delete_FullMethodName, true},
		{"dir.com", storev1.StoreService_Push_FullMethodName, true},
		{"dir.com", routingv1.RoutingService_Publish_FullMethodName, true},

		// anyone else: only pull/lookup/sync
		{"other.com", storev1.StoreService_Pull_FullMethodName, true},
		{"other.com", storev1.StoreService_Lookup_FullMethodName, true},
		{"other.com", storev1.SyncService_RequestRegistryCredentials_FullMethodName, true},
		{"other.com", storev1.StoreService_Push_FullMethodName, false},
		{"other.com", routingv1.RoutingService_Publish_FullMethodName, false},
	}

	for _, tt := range tests {
		allowed, err := authz.Authorize(context.Background(), tt.trustDomain, tt.apiMethod)
		if err != nil {
			t.Errorf("Authorize() error: %v", err)
		}
		if allowed != tt.allow {
			t.Errorf("Authorize(%q, %q) = %v, want %v", tt.trustDomain, tt.apiMethod, allowed, tt.allow)
		}
	}
}
