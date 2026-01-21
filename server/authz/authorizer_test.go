// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package authz

import (
	"testing"

	routingv1 "github.com/agntcy/dir/api/routing/v1"
	storev1 "github.com/agntcy/dir/api/store/v1"
	"github.com/agntcy/dir/server/authz/config"
)

func TestAuthorizer(t *testing.T) {
	defaultAuthz, err := NewAuthorizer(config.Config{
		TrustDomain: "dir.com",
	})
	if err != nil {
		t.Fatalf("failed to create Casbin authorizer: %v", err)
	}

	customAuthz, err := NewAuthorizer(config.Config{
		TrustDomain: "dir.com",
		Policies: map[string][]string{
			"dir.com": {
				storev1.StoreService_Delete_FullMethodName,
				storev1.StoreService_Push_FullMethodName,
			},
			"other.com": {
				storev1.StoreService_Lookup_FullMethodName,
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to create Casbin authorizer: %v", err)
	}

	tests := []struct {
		authorizer  *Authorizer
		trustDomain string
		apiMethod   string
		allow       bool
	}{
		// dir.com: all ops allowed
		{defaultAuthz, "dir.com", storev1.StoreService_Delete_FullMethodName, true},
		{defaultAuthz, "dir.com", storev1.StoreService_Push_FullMethodName, true},
		{defaultAuthz, "dir.com", routingv1.RoutingService_Publish_FullMethodName, true},

		// anyone else: only pull
		{defaultAuthz, "other.com", storev1.StoreService_Pull_FullMethodName, true},
		{defaultAuthz, "other.com", storev1.StoreService_Lookup_FullMethodName, false},
		{defaultAuthz, "other.com", storev1.SyncService_RequestRegistryCredentials_FullMethodName, false},
		{defaultAuthz, "other.com", storev1.StoreService_Push_FullMethodName, false},
		{defaultAuthz, "other.com", routingv1.RoutingService_Publish_FullMethodName, false},

		// custom policies
		{customAuthz, "dir.com", storev1.StoreService_Delete_FullMethodName, true},
		{customAuthz, "dir.com", storev1.StoreService_Push_FullMethodName, true},
		{customAuthz, "dir.com", routingv1.RoutingService_Publish_FullMethodName, false},

		{customAuthz, "other.com", storev1.StoreService_Pull_FullMethodName, false},
		{customAuthz, "other.com", storev1.StoreService_Lookup_FullMethodName, true},
		{customAuthz, "other.com", storev1.SyncService_RequestRegistryCredentials_FullMethodName, false},
		{customAuthz, "other.com", storev1.StoreService_Push_FullMethodName, false},
		{customAuthz, "other.com", routingv1.RoutingService_Publish_FullMethodName, false},
	}

	for _, tt := range tests {
		allowed, err := tt.authorizer.Authorize(tt.trustDomain, tt.apiMethod)
		if err != nil {
			t.Errorf("Authorize() error: %v", err)
		}

		if allowed != tt.allow {
			t.Errorf("Authorize(%q, %q) = %v, want %v", tt.trustDomain, tt.apiMethod, allowed, tt.allow)
		}
	}
}
