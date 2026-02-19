// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package authz

import (
	"testing"

	storev1 "github.com/agntcy/dir/api/store/v1"
	"github.com/agntcy/dir/server/authz/config"
)

func TestAuthorizer(t *testing.T) {
	allowAllAuthz, err := NewAuthorizer(config.Config{
		EnforcerPolicyFilePath: "./testdata/allow_all_policies.csv",
	})
	if err != nil {
		t.Fatalf("failed to create Casbin authorizer: %v", err)
	}

	externalsOnlyAuthz, err := NewAuthorizer(config.Config{
		EnforcerPolicyFilePath: "./testdata/externals_only_policies.csv",
	})
	if err != nil {
		t.Fatalf("failed to create Casbin authorizer: %v", err)
	}

	externalPullOnlyAuthz, err := NewAuthorizer(config.Config{
		EnforcerPolicyFilePath: "./testdata/external_pull_only_policies.csv",
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
		{allowAllAuthz, "allow_all_dir.com", storev1.StoreService_Push_FullMethodName, true},
		{allowAllAuthz, "allow_all_dir.com", storev1.StoreService_Pull_FullMethodName, true},
		{allowAllAuthz, "allow_all_dir.com", storev1.StoreService_Lookup_FullMethodName, true},
		{allowAllAuthz, "allow_all_dir.com", storev1.StoreService_Delete_FullMethodName, true},
		{allowAllAuthz, "allow_all_dir.com", storev1.StoreService_PushReferrer_FullMethodName, true},
		{allowAllAuthz, "allow_all_dir.com", storev1.StoreService_PullReferrer_FullMethodName, true},
		{allowAllAuthz, "allow_all_dir.com", storev1.SyncService_CreateSync_FullMethodName, true},
		{allowAllAuthz, "allow_all_dir.com", storev1.SyncService_ListSyncs_FullMethodName, true},
		{allowAllAuthz, "allow_all_dir.com", storev1.SyncService_GetSync_FullMethodName, true},
		{allowAllAuthz, "allow_all_dir.com", storev1.SyncService_RequestRegistryCredentials_FullMethodName, true},

		{externalsOnlyAuthz, "externals_only_dir.com", storev1.StoreService_Push_FullMethodName, false},
		{externalsOnlyAuthz, "externals_only_dir.com", storev1.StoreService_Pull_FullMethodName, false},
		{externalsOnlyAuthz, "externals_only_dir.com", storev1.StoreService_Lookup_FullMethodName, false},
		{externalsOnlyAuthz, "externals_only_dir.com", storev1.StoreService_Delete_FullMethodName, false},
		{externalsOnlyAuthz, "externals_only_dir.com", storev1.StoreService_PushReferrer_FullMethodName, false},
		{externalsOnlyAuthz, "externals_only_dir.com", storev1.StoreService_PullReferrer_FullMethodName, false},
		{externalsOnlyAuthz, "externals_only_dir.com", storev1.SyncService_CreateSync_FullMethodName, false},
		{externalsOnlyAuthz, "externals_only_dir.com", storev1.SyncService_ListSyncs_FullMethodName, false},
		{externalsOnlyAuthz, "externals_only_dir.com", storev1.SyncService_GetSync_FullMethodName, false},
		{externalsOnlyAuthz, "externals_only_dir.com", storev1.SyncService_RequestRegistryCredentials_FullMethodName, false},
		{externalsOnlyAuthz, "externals_only_other.com", storev1.StoreService_Push_FullMethodName, true},
		{externalsOnlyAuthz, "externals_only_other.com", storev1.StoreService_Pull_FullMethodName, true},
		{externalsOnlyAuthz, "externals_only_other.com", storev1.StoreService_Lookup_FullMethodName, true},
		{externalsOnlyAuthz, "externals_only_other.com", storev1.StoreService_Delete_FullMethodName, true},
		{externalsOnlyAuthz, "externals_only_other.com", storev1.StoreService_PushReferrer_FullMethodName, true},
		{externalsOnlyAuthz, "externals_only_other.com", storev1.StoreService_PullReferrer_FullMethodName, true},
		{externalsOnlyAuthz, "externals_only_other.com", storev1.SyncService_CreateSync_FullMethodName, false},
		{externalsOnlyAuthz, "externals_only_other.com", storev1.SyncService_ListSyncs_FullMethodName, false},
		{externalsOnlyAuthz, "externals_only_other.com", storev1.SyncService_GetSync_FullMethodName, false},
		{externalsOnlyAuthz, "externals_only_other.com", storev1.SyncService_RequestRegistryCredentials_FullMethodName, false},
		{externalsOnlyAuthz, "externals_only_other2.com", storev1.StoreService_Push_FullMethodName, false},
		{externalsOnlyAuthz, "externals_only_other2.com", storev1.StoreService_Pull_FullMethodName, false},
		{externalsOnlyAuthz, "externals_only_other2.com", storev1.StoreService_Lookup_FullMethodName, false},
		{externalsOnlyAuthz, "externals_only_other2.com", storev1.StoreService_Delete_FullMethodName, false},
		{externalsOnlyAuthz, "externals_only_other2.com", storev1.StoreService_PushReferrer_FullMethodName, false},
		{externalsOnlyAuthz, "externals_only_other2.com", storev1.StoreService_PullReferrer_FullMethodName, false},
		{externalsOnlyAuthz, "externals_only_other2.com", storev1.SyncService_CreateSync_FullMethodName, true},
		{externalsOnlyAuthz, "externals_only_other2.com", storev1.SyncService_ListSyncs_FullMethodName, true},
		{externalsOnlyAuthz, "externals_only_other2.com", storev1.SyncService_GetSync_FullMethodName, true},
		{externalsOnlyAuthz, "externals_only_other2.com", storev1.SyncService_RequestRegistryCredentials_FullMethodName, true},

		{externalPullOnlyAuthz, "external_pull_only_dir.com", storev1.StoreService_Push_FullMethodName, true},
		{externalPullOnlyAuthz, "external_pull_only_dir.com", storev1.StoreService_Pull_FullMethodName, true},
		{externalPullOnlyAuthz, "external_pull_only_dir.com", storev1.StoreService_Lookup_FullMethodName, true},
		{externalPullOnlyAuthz, "external_pull_only_dir.com", storev1.StoreService_Delete_FullMethodName, true},
		{externalPullOnlyAuthz, "external_pull_only_dir.com", storev1.StoreService_PushReferrer_FullMethodName, true},
		{externalPullOnlyAuthz, "external_pull_only_dir.com", storev1.StoreService_PullReferrer_FullMethodName, true},
		{externalPullOnlyAuthz, "external_pull_only_dir.com", storev1.SyncService_CreateSync_FullMethodName, true},
		{externalPullOnlyAuthz, "external_pull_only_dir.com", storev1.SyncService_ListSyncs_FullMethodName, true},
		{externalPullOnlyAuthz, "external_pull_only_dir.com", storev1.SyncService_GetSync_FullMethodName, true},
		{externalPullOnlyAuthz, "external_pull_only_dir.com", storev1.SyncService_RequestRegistryCredentials_FullMethodName, true},
		{externalPullOnlyAuthz, "external_pull_only_other.com", storev1.StoreService_Push_FullMethodName, false},
		{externalPullOnlyAuthz, "external_pull_only_other.com", storev1.StoreService_Pull_FullMethodName, true},
		{externalPullOnlyAuthz, "external_pull_only_other.com", storev1.StoreService_Lookup_FullMethodName, false},
		{externalPullOnlyAuthz, "external_pull_only_other.com", storev1.StoreService_Delete_FullMethodName, false},
		{externalPullOnlyAuthz, "external_pull_only_other.com", storev1.StoreService_PushReferrer_FullMethodName, false},
		{externalPullOnlyAuthz, "external_pull_only_other.com", storev1.StoreService_PullReferrer_FullMethodName, true},
		{externalPullOnlyAuthz, "external_pull_only_other.com", storev1.SyncService_CreateSync_FullMethodName, false},
		{externalPullOnlyAuthz, "external_pull_only_other.com", storev1.SyncService_ListSyncs_FullMethodName, false},
		{externalPullOnlyAuthz, "external_pull_only_other.com", storev1.SyncService_GetSync_FullMethodName, false},
		{externalPullOnlyAuthz, "external_pull_only_other.com", storev1.SyncService_RequestRegistryCredentials_FullMethodName, false},
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
