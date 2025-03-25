package corev1alpha1

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestObjectRef_CIDConversion(t *testing.T) {
	testCases := []struct {
		name    string
		objType string
		digest  string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid raw object",
			objType: ObjectType_OBJECT_TYPE_RAW.String(),
			digest:  "sha256:1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
			wantErr: false,
		},
		{
			name:    "valid agent object",
			objType: ObjectType_OBJECT_TYPE_AGENT.String(),
			digest:  "sha256:1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
			wantErr: false,
		},
		{
			name:    "invalid digest format",
			objType: ObjectType_OBJECT_TYPE_RAW.String(),
			digest:  "invalid-digest",
			wantErr: true,
			errMsg:  "invalid digest format",
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
			cid, err := orig.GetCID()
			if tc.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.errMsg)
				return
			}
			assert.NoError(t, err)

			// Convert back from CID
			converted := &ObjectRef{}
			err = converted.FromCID(cid)
			assert.NoError(t, err)

			// Verify the round-trip conversion
			assert.Equal(t, orig.Type, converted.Type)
			assert.Equal(t, orig.Digest, converted.Digest)
		})
	}
}
