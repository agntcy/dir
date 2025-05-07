// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:testifylint
package corev1alpha1

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestObjectRef_CIDConversion(t *testing.T) {
	testCases := []struct {
		name     string
		objType  string
		digest   string
		shortRef string
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "valid raw object",
			objType:  ObjectType_OBJECT_TYPE_RAW.String(),
			digest:   "sha256:1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
			shortRef: "QmVz2CRrxr7uyYtoQo1Qszvihd7JY1w7Yyth5F5AgmXPG6",
			wantErr:  false,
		},
		{
			name:     "valid agent object",
			objType:  ObjectType_OBJECT_TYPE_AGENT.String(),
			digest:   "sha256:34567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef12",
			shortRef: "QmRegDXGWPHqoXkDB7G9ANPXVaucvFS8ygqvQasRf2qXny",
			wantErr:  false,
		},
		{
			name:     "invalid digest format",
			objType:  ObjectType_OBJECT_TYPE_RAW.String(),
			digest:   "invalid-digest",
			shortRef: "invalid-ref",
			wantErr:  true,
			errMsg:   "invalid digest format",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create initial ObjectRef
			orig := &ObjectRef{
				Type:   tc.objType,
				Digest: tc.digest,
			}

			// Convert to CID
			origCid, err := orig.GetCID()
			if tc.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.errMsg)

				return
			}

			assert.NoError(t, err)

			// Convert back from CID
			converted := &ObjectRef{}
			err = converted.FromCID(origCid)
			assert.NoError(t, err)

			// Verify the round-trip conversion
			assert.Equal(t, orig.GetType(), converted.GetType())
			assert.Equal(t, orig.GetDigest(), converted.GetDigest())
			assert.Equal(t, orig.GetShortRef(), tc.shortRef)
		})
	}
}
