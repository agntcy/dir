// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package labels

import (
	"errors"
	"fmt"
	"strings"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/server/types/adapters"
	"github.com/agntcy/dir/utils/logging"
)

var labelLogger = logging.Logger("labels")

// GetLabels extracts all labels from a record across all supported namespaces.
// This is a pure function that can be used by both local and remote routing operations.
//
// The function extracts labels from:
// - Skills: /skills/<skill_name>
// - Domains: /domains/<domain_name> (from extensions with domain schema prefix)
// - Features: /features/<feature_name> (from extensions with features schema prefix)
// - Locators: /locators/<locator_type>
//
// Returns a slice of typed Labels with namespace prefixes for type safety.
func GetLabels(record *corev1.Record) []Label {
	// Use adapter pattern to get version-agnostic access to record data
	adapter := adapters.NewRecordAdapter(record)

	recordData := adapter.GetRecordData()
	if recordData == nil {
		labelLogger.Error("failed to get record data")

		return nil
	}

	var labels []Label //nolint:prealloc // Cannot reasonably pre-allocate - depends on variable record content

	// Extract record skills
	for _, skill := range recordData.GetSkills() {
		skillLabel := Label(LabelTypeSkill.Prefix() + skill.GetName())
		labels = append(labels, skillLabel)
	}

	// Extract record domains from extensions
	for _, ext := range recordData.GetExtensions() {
		if strings.HasPrefix(ext.GetName(), DomainSchemaPrefix) {
			domain := ext.GetName()[len(DomainSchemaPrefix):]
			domainLabel := Label(LabelTypeDomain.Prefix() + domain)
			labels = append(labels, domainLabel)
		}
	}

	// Extract record features from extensions
	for _, ext := range recordData.GetExtensions() {
		if strings.HasPrefix(ext.GetName(), FeaturesSchemaPrefix) {
			feature := ext.GetName()[len(FeaturesSchemaPrefix):]
			featureLabel := Label(LabelTypeFeature.Prefix() + feature)
			labels = append(labels, featureLabel)
		}
	}

	// Extract record locators
	for _, locator := range recordData.GetLocators() {
		locatorLabel := Label(LabelTypeLocator.Prefix() + locator.GetType())
		labels = append(labels, locatorLabel)
	}

	return labels
}

// Example: Label("/skills/AI/ML") → "/skills/AI/ML/CID123/Peer1".
func BuildEnhancedLabelKey(label Label, cid, peerID string) string {
	return fmt.Sprintf("%s/%s/%s", label.String(), cid, peerID)
}

// BuildEnhancedLabelKeyFromString creates a key from a string label (for backward compatibility).
// This is used during migration and for legacy code.
func BuildEnhancedLabelKeyFromString(label, cid, peerID string) string {
	return fmt.Sprintf("%s/%s/%s", label, cid, peerID)
}

// Example: "/skills/AI/ML/CID123/Peer1" → (Label("/skills/AI/ML"), "CID123", "Peer1", nil).
func ParseEnhancedLabelKey(key string) (Label, string, string, error) {
	labelStr, cid, peerID, err := parseEnhancedLabelKeyInternal(key)
	if err != nil {
		return Label(""), "", "", err
	}

	return Label(labelStr), cid, peerID, nil
}

// ParseEnhancedLabelKeyToString parses a key and returns string components (for backward compatibility).
// This is used during migration and for legacy code.
func ParseEnhancedLabelKeyToString(key string) (string, string, string, error) {
	return parseEnhancedLabelKeyInternal(key)
}

// parseEnhancedLabelKeyInternal contains the actual parsing logic.
// This is shared by both the Label-returning and string-returning functions.
func parseEnhancedLabelKeyInternal(key string) (string, string, string, error) {
	if !strings.HasPrefix(key, "/") {
		return "", "", "", errors.New("key must start with /")
	}

	parts := strings.Split(key, "/")
	if len(parts) < MinLabelKeyParts {
		return "", "", "", errors.New("key must have at least namespace/path/CID/PeerID")
	}

	// Extract PeerID (last part) and CID (second to last part)
	peerID := parts[len(parts)-1]
	cid := parts[len(parts)-2]

	// Extract label (everything except the last two parts)
	labelParts := parts[1 : len(parts)-2] // Skip empty first part and last two parts
	label := "/" + strings.Join(labelParts, "/")

	return label, cid, peerID, nil
}

// ExtractPeerIDFromKey extracts just the PeerID from a self-descriptive key.
func ExtractPeerIDFromKey(key string) string {
	parts := strings.Split(key, "/")
	if len(parts) < MinLabelKeyParts {
		return ""
	}

	return parts[len(parts)-1]
}

// ExtractCIDFromKey extracts just the CID from a self-descriptive key.
func ExtractCIDFromKey(key string) string {
	parts := strings.Split(key, "/")
	if len(parts) < MinLabelKeyParts {
		return ""
	}

	return parts[len(parts)-2]
}

// IsLocalKey checks if a key belongs to the given local peer ID.
func IsLocalKey(key, localPeerID string) bool {
	return ExtractPeerIDFromKey(key) == localPeerID
}

// IsValidLabelKey checks if a key starts with any valid label type prefix.
// Returns true if the key starts with /skills/, /domains/, /features/, or /locators/.
func IsValidLabelKey(key string) bool {
	for _, labelType := range AllLabelTypes() {
		if strings.HasPrefix(key, labelType.Prefix()) {
			return true
		}
	}

	return false
}

// GetLabelTypeFromKey extracts the label type from a key.
// Returns the label type and true if found, or LabelTypeUnknown and false if not found.
func GetLabelTypeFromKey(key string) (LabelType, bool) {
	for _, labelType := range AllLabelTypes() {
		if strings.HasPrefix(key, labelType.Prefix()) {
			return labelType, true
		}
	}

	return LabelTypeUnknown, false
}
