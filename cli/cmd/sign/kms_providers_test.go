// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package sign

import (
	"slices"
	"testing"

	"github.com/sigstore/sigstore/pkg/signature/kms"
)

func TestKMSProvidersRegistered(t *testing.T) {
	t.Parallel()

	providers := kms.SupportedProviders()
	slices.Sort(providers)

	for _, expected := range []string{
		"awskms://",
		"azurekms://",
		"gcpkms://",
		"hashivault://",
	} {
		if !slices.Contains(providers, expected) {
			t.Errorf("registered KMS providers = %v, want %q", providers, expected)
		}
	}
}
