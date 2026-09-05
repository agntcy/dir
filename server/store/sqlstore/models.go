// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package sqlstore

// Record holds one content-addressed record.
//
// CanonicalBytes is the authoritative content and RecordCID is derived from it,
// so the pair must never be updated in place. The remaining columns are
// denormalised from the record at push time so Lookup can answer without
// decoding the payload.
type Record struct {
	RecordCID      string `gorm:"column:record_cid;primaryKey;not null"`
	CanonicalBytes []byte `gorm:"column:canonical_bytes;not null"`

	Name          string            `gorm:"column:name;index"`
	Version       string            `gorm:"column:version"`
	SchemaVersion string            `gorm:"column:schema_version"`
	OASFVersion   string            `gorm:"column:oasf_version"`
	OASFCreatedAt string            `gorm:"column:oasf_created_at"`
	PreviousCID   string            `gorm:"column:previous_cid"`
	Annotations   map[string]string `gorm:"column:annotations;serializer:json"`

	Referrers []Referrer `gorm:"foreignKey:SubjectCID;references:RecordCID;constraint:OnDelete:CASCADE"`
}

func (Record) TableName() string { return "store_records" }

// Referrer is an artifact attached to a record, keyed by its own CID and the
// CID of its subject. The same referrer may be attached to more than one
// subject, hence the composite key. The foreign key on SubjectCID is what makes
// record deletion cascade to referrers.
type Referrer struct {
	ReferrerCID string `gorm:"column:referrer_cid;primaryKey;not null"`
	SubjectCID  string `gorm:"column:subject_cid;primaryKey;not null;index:idx_referrer_subject_type,priority:1"`
	Type        string `gorm:"column:type;index:idx_referrer_subject_type,priority:2"`
	Payload     []byte `gorm:"column:payload;not null"`
}

func (Referrer) TableName() string { return "store_referrers" }
