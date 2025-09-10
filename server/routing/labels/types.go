// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package labels provides unified label types and operations for the routing system.
// This package serves as the single source of truth for all label-related functionality,
// eliminating the previous split between validators/namespaces and routing/labels.
package labels

import (
	"errors"
	"strings"
	"time"
)

// LabelType represents the category of a label based on its namespace.
// Using string type for natural representation and direct DHT integration.
type LabelType string

const (
	LabelTypeUnknown LabelType = ""
	LabelTypeSkill   LabelType = "skills"
	LabelTypeDomain  LabelType = "domains"
	LabelTypeFeature LabelType = "features"
	LabelTypeLocator LabelType = "locators"
)

// String returns the string representation of the label type.
// This is used for DHT validation, logging, and debugging.
func (lt LabelType) String() string {
	return string(lt)
}

// Example: LabelTypeSkill.Prefix() returns "/skills/".
func (lt LabelType) Prefix() string {
	if lt == LabelTypeUnknown {
		return ""
	}

	return "/" + string(lt) + "/"
}

// IsValid checks if the label type is one of the supported types.
func (lt LabelType) IsValid() bool {
	switch lt {
	case LabelTypeSkill, LabelTypeDomain, LabelTypeFeature, LabelTypeLocator:
		return true
	case LabelTypeUnknown:
		return false
	default:
		return false
	}
}

// AllLabelTypes returns all supported label types.
func AllLabelTypes() []LabelType {
	return []LabelType{LabelTypeSkill, LabelTypeDomain, LabelTypeFeature, LabelTypeLocator}
}

// ParseLabelType converts a string to LabelType if valid.
func ParseLabelType(s string) (LabelType, bool) {
	lt := LabelType(s)
	if lt.IsValid() {
		return lt, true
	}

	return LabelTypeUnknown, false
}

// Label represents a typed label with namespace awareness.
// This provides type safety and eliminates string-based operations throughout the routing system.
type Label string

// String returns the string representation of the label.
// This is used for storage, logging, and API boundary conversions.
func (l Label) String() string {
	return string(l)
}

// Bytes returns the byte representation for efficient storage operations.
// This eliminates the need for string conversions in datastore operations.
func (l Label) Bytes() []byte {
	return []byte(l)
}

// Type returns the type of the label based on its namespace prefix.
// This enables efficient type-based filtering without complex lookups.
func (l Label) Type() LabelType {
	s := string(l)

	switch {
	case strings.HasPrefix(s, LabelTypeSkill.Prefix()):
		return LabelTypeSkill
	case strings.HasPrefix(s, LabelTypeDomain.Prefix()):
		return LabelTypeDomain
	case strings.HasPrefix(s, LabelTypeFeature.Prefix()):
		return LabelTypeFeature
	case strings.HasPrefix(s, LabelTypeLocator.Prefix()):
		return LabelTypeLocator
	default:
		return LabelTypeUnknown
	}
}

// Namespace returns the namespace prefix of the label.
// For example, Label("/skills/AI") returns "/skills/".
func (l Label) Namespace() string {
	return l.Type().Prefix()
}

// Value returns the label value without the namespace prefix.
// For example, Label("/skills/AI/ML") returns "AI/ML".
func (l Label) Value() string {
	namespace := l.Namespace()
	if namespace == "" {
		return string(l)
	}

	return strings.TrimPrefix(string(l), namespace)
}

// The PeerID and CID are stored in the key structure: /skills/AI/CID123/Peer1.
type LabelMetadata struct {
	Timestamp time.Time `json:"timestamp"` // When label was first announced
	LastSeen  time.Time `json:"last_seen"` // When label was last seen/refreshed
}

// Validate checks if the metadata is valid and all required fields are properly set.
func (m *LabelMetadata) Validate() error {
	if m.Timestamp.IsZero() {
		return errors.New("timestamp cannot be zero")
	}

	if m.LastSeen.IsZero() {
		return errors.New("last seen timestamp cannot be zero")
	}

	if m.LastSeen.Before(m.Timestamp) {
		return errors.New("last seen cannot be before creation timestamp")
	}

	return nil
}

// IsStale checks if the label is older than the given maximum age duration.
func (m *LabelMetadata) IsStale(maxAge time.Duration) bool {
	return time.Since(m.LastSeen) > maxAge
}

// Age returns how long ago the label was last seen.
func (m *LabelMetadata) Age() time.Duration {
	return time.Since(m.LastSeen)
}

// Update refreshes the LastSeen timestamp to the current time.
func (m *LabelMetadata) Update() {
	m.LastSeen = time.Now()
}

// Constants for label validation and processing.
const (
	// Enhanced format: /type/label/CID/PeerID splits into ["", "type", "label", "CID", "PeerID"] = 5 parts.
	MinLabelKeyParts = 5

	// Schema prefixes for extracting labels from record extensions.
	DomainSchemaPrefix   = "schema.oasf.agntcy.org/domains/"
	FeaturesSchemaPrefix = "schema.oasf.agntcy.org/features/"
)
