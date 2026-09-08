// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package v1_test

import (
	"testing"

	oasfv1alpha1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/agntcy/oasf/types/v1alpha1"
	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/stretchr/testify/assert"
)

func TestRecord_GetIdentity_GetOwner(t *testing.T) {
	record := corev1.New(&oasfv1alpha1.Record{
		Name:          "finance-agent",
		SchemaVersion: "v0.5.0",
		Version:       "1.0.0",
		Annotations: map[string]string{
			corev1.AnnotationKeyIdentity:  "did:web:acme.com:agents:finance",
			corev1.AnnotationKeyOwner:     "spiffe://acme.com/org",
			corev1.AnnotationKeyOwnerType: "spiffe",
		},
	})

	assert.Equal(t, "did:web:acme.com:agents:finance", record.GetIdentity())
	assert.Equal(t, "did", record.GetIdentityType(), "identity-type should be inferred from the did: prefix")
	assert.Equal(t, "spiffe://acme.com/org", record.GetOwner())
	assert.Equal(t, "spiffe", record.GetOwnerType(), "owner-type should use the explicit annotation override")
}

func TestRecord_GetIdentity_GetOwner_Absent(t *testing.T) {
	record := corev1.New(&oasfv1alpha1.Record{
		Name:          "no-identity-agent",
		SchemaVersion: "v0.5.0",
		Version:       "1.0.0",
	})

	assert.Empty(t, record.GetIdentity())
	assert.Empty(t, record.GetOwner())
	assert.Empty(t, record.GetIdentityType())
	assert.Empty(t, record.GetOwnerType())
}

func TestRecord_GetIdentityType_InferredFromScheme(t *testing.T) {
	tests := []struct {
		name     string
		identity string
		want     string
	}{
		{name: "did", identity: "did:web:acme.com", want: "did"},
		{name: "spiffe", identity: "spiffe://acme.com/agent", want: "spiffe"},
		{name: "https", identity: "https://acme.com/agent", want: "https"},
		{name: "http", identity: "http://acme.com/agent", want: "https"},
		{name: "bare domain falls back to dns", identity: "acme.com", want: "dns"},
		{name: "explicit dns prefix", identity: "dns:acme.com", want: "dns"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			record := corev1.New(&oasfv1alpha1.Record{
				Name:          "test-agent",
				SchemaVersion: "v0.5.0",
				Annotations: map[string]string{
					corev1.AnnotationKeyIdentity: tt.identity,
				},
			})

			assert.Equal(t, tt.want, record.GetIdentityType())
		})
	}
}

func TestRecord_GetCid_StableWithIdentityAnnotations(t *testing.T) {
	base := func() *oasfv1alpha1.Record {
		return &oasfv1alpha1.Record{
			Name:          "finance-agent",
			SchemaVersion: "v0.5.0",
			Version:       "1.0.0",
			Annotations: map[string]string{
				corev1.AnnotationKeyIdentity: "did:web:acme.com:agents:finance",
				corev1.AnnotationKeyOwner:    "did:web:acme.com",
			},
		}
	}

	record1 := corev1.New(base())
	record2 := corev1.New(base())

	cid1 := record1.GetCid()
	cid2 := record2.GetCid()

	assert.NotEmpty(t, cid1)
	assert.Equal(t, cid1, cid2, "identical identity/owner annotations must produce the same CID")
}

func TestRecord_GetCid_ChangesWhenIdentityAnnotationChanges(t *testing.T) {
	original := corev1.New(&oasfv1alpha1.Record{
		Name:          "finance-agent",
		SchemaVersion: "v0.5.0",
		Version:       "1.0.0",
		Annotations: map[string]string{
			corev1.AnnotationKeyIdentity: "did:web:acme.com:agents:finance",
		},
	})

	tampered := corev1.New(&oasfv1alpha1.Record{
		Name:          "finance-agent",
		SchemaVersion: "v0.5.0",
		Version:       "1.0.0",
		Annotations: map[string]string{
			corev1.AnnotationKeyIdentity: "did:web:attacker.com:agents:finance",
		},
	})

	assert.NotEqual(t, original.GetCid(), tampered.GetCid(), "changing the identity annotation must change the record's CID")
}
