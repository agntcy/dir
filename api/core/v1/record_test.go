// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package corev1

import (
	"testing"

	objectsv1 "github.com/agntcy/dir/api/objects/v1"
	objectsv3 "github.com/agntcy/dir/api/objects/v3"
	"github.com/stretchr/testify/assert"
)

func TestRecord_GetCid(t *testing.T) {
	tests := []struct {
		name    string
		record  *Record
		want    string
		wantErr bool
	}{
		{
			name: "v0.3.1 agent record",
			record: &Record{
				Data: &Record_V1{
					V1: &objectsv1.Agent{
						Name:          "test-agent",
						SchemaVersion: "v0.3.1",
						Description:   "A test agent",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "v0.5.0 record",
			record: &Record{
				Data: &Record_V3{
					V3: &objectsv3.Record{
						Name:          "test-agent-v2",
						SchemaVersion: "v0.5.0",
						Description:   "A test agent in v0.5.0 record",
						Version:       "1.0.0",
						Extensions: []*objectsv3.Extension{
							{
								Name:    "test-extension",
								Version: "1.0.0",
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name:    "nil record",
			record:  nil,
			wantErr: true,
		},
		{
			name:    "empty record",
			record:  &Record{},
			wantErr: true, // Empty record should fail - no OASF data to marshal
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cid := tt.record.GetCid()

			if tt.wantErr {
				assert.Empty(t, cid)

				return
			}

			assert.NotEmpty(t, cid)

			// CID should be consistent - calling it again should return the same value.
			cid2 := tt.record.GetCid()
			assert.Equal(t, cid, cid2, "CID should be deterministic")

			// CID should start with the CIDv1 prefix.
			assert.Greater(t, len(cid), 10, "CID should be a reasonable length")
		})
	}
}

func TestRecord_GetCid_Consistency(t *testing.T) {
	// Create two identical v0.3.1 records.
	agent := &objectsv1.Agent{
		Name:          "test-agent",
		SchemaVersion: "v0.3.1",
		Description:   "A test agent",
	}

	record1 := &Record{
		Data: &Record_V1{
			V1: agent,
		},
	}

	record2 := &Record{
		Data: &Record_V1{
			V1: &objectsv1.Agent{
				Name:          "test-agent",
				SchemaVersion: "v0.3.1",
				Description:   "A test agent",
			},
		},
	}

	// Both records should have the same CID.
	cid1 := record1.GetCid()
	cid2 := record2.GetCid()

	assert.Equal(t, cid1, cid2, "Identical v0.3.1 records should have identical CIDs")
}

func TestRecord_GetCid_V0_5_0_Consistency(t *testing.T) {
	// Create two identical v0.5.0 records.
	v0_5_0Record1 := &Record{
		Data: &Record_V3{
			V3: &objectsv3.Record{
				Name:          "test-agent-v2",
				SchemaVersion: "v0.5.0",
				Description:   "A test agent in v0.5.0 record",
				Version:       "1.0.0",
				Extensions: []*objectsv3.Extension{
					{
						Name:    "test-extension",
						Version: "1.0.0",
					},
				},
			},
		},
	}

	v0_5_0Record2 := &Record{
		Data: &Record_V3{
			V3: &objectsv3.Record{
				Name:          "test-agent-v2",
				SchemaVersion: "v0.5.0",
				Description:   "A test agent in v0.5.0 record",
				Version:       "1.0.0",
				Extensions: []*objectsv3.Extension{
					{
						Name:    "test-extension",
						Version: "1.0.0",
					},
				},
			},
		},
	}

	// Both records should have the same CID.
	cid1 := v0_5_0Record1.GetCid()
	cid2 := v0_5_0Record2.GetCid()

	assert.Equal(t, cid1, cid2, "Identical v0.5.0 records should have identical CIDs")
}

func TestRecord_GetCid_CrossVersion_Difference(t *testing.T) {
	// Create similar but different version records - they should have different CIDs.
	v0_3_1Record := &Record{
		Data: &Record_V1{
			V1: &objectsv1.Agent{
				Name:          "test-agent",
				SchemaVersion: "v0.3.1",
				Description:   "A test agent",
			},
		},
	}

	v0_5_0Record := &Record{
		Data: &Record_V3{
			V3: &objectsv3.Record{
				Name:          "test-agent",
				SchemaVersion: "v0.5.0",
				Description:   "A test agent",
				Version:       "1.0.0",
			},
		},
	}

	cid1 := v0_3_1Record.GetCid()
	cid2 := v0_5_0Record.GetCid()

	assert.NotEqual(t, cid1, cid2, "Different record versions should have different CIDs")
}

func TestRecord_MustGetCid(t *testing.T) {
	record := &Record{
		Data: &Record_V1{
			V1: &objectsv1.Agent{
				Name:          "test-agent",
				SchemaVersion: "v0.3.1",
				Description:   "A test agent",
			},
		},
	}

	// MustGetCid should not panic for valid record.
	assert.NotPanics(t, func() {
		cid := record.MustGetCid()
		assert.NotEmpty(t, cid)
	})

	// Test with v0.5.0 record.
	v0_5_0Record := &Record{
		Data: &Record_V3{
			V3: &objectsv3.Record{
				Name:          "test-agent-v2",
				SchemaVersion: "v0.5.0",
				Description:   "A test agent in v0.5.0 record",
				Version:       "1.0.0",
			},
		},
	}

	assert.NotPanics(t, func() {
		cid := v0_5_0Record.MustGetCid()
		assert.NotEmpty(t, cid)
	})

	// MustGetCid should panic for nil record.
	var nilRecord *Record

	assert.Panics(t, func() {
		nilRecord.MustGetCid()
	})

	// MustGetCid should panic for empty record (no OASF data).
	emptyRecord := &Record{}

	assert.Panics(t, func() {
		emptyRecord.MustGetCid()
	})
}
