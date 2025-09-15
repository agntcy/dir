// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package routing

import (
	"errors"
	"fmt"
	"strings"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/server/types/adapters"
	"github.com/agntcy/dir/server/types/labels"
)

// Key manipulation utilities for routing operations.
// These functions handle the enhanced label key format: /namespace/value/CID/PeerID

// Example: Label("/skills/AI/ML") → "/skills/AI/ML/CID123/Peer1".
func BuildEnhancedLabelKey(label labels.Label, cid, peerID string) string {
	return fmt.Sprintf("%s/%s/%s", label.String(), cid, peerID)
}

// Example: "/skills/AI/ML/CID123/Peer1" → (Label("/skills/AI/ML"), "CID123", "Peer1", nil).
func ParseEnhancedLabelKey(key string) (labels.Label, string, string, error) {
	labelStr, cid, peerID, err := parseEnhancedLabelKeyInternal(key)
	if err != nil {
		return labels.Label(""), "", "", err
	}

	return labels.Label(labelStr), cid, peerID, nil
}

// parseEnhancedLabelKeyInternal contains the actual parsing logic.
// This is used internally by ParseEnhancedLabelKey.
func parseEnhancedLabelKeyInternal(key string) (string, string, string, error) {
	if !strings.HasPrefix(key, "/") {
		return "", "", "", errors.New("key must start with /")
	}

	parts := strings.Split(key, "/")
	if len(parts) < labels.MinLabelKeyParts {
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
	if len(parts) < labels.MinLabelKeyParts {
		return ""
	}

	return parts[len(parts)-1]
}

// IsValidLabelKey checks if a key starts with any valid label type prefix.
// Returns true if the key starts with /skills/, /domains/, /features/, or /locators/.
func IsValidLabelKey(key string) bool {
	for _, labelType := range labels.AllLabelTypes() {
		if strings.HasPrefix(key, labelType.Prefix()) {
			return true
		}
	}

	return false
}

// GetLabelTypeFromKey extracts the label type from a key.
// Returns the label type and true if found, or LabelTypeUnknown and false if not found.
func GetLabelTypeFromKey(key string) (labels.LabelType, bool) {
	for _, labelType := range labels.AllLabelTypes() {
		if strings.HasPrefix(key, labelType.Prefix()) {
			return labelType, true
		}
	}

	return labels.LabelTypeUnknown, false
}

// GetLabelsFromRecord extracts labels using the new LabelProvider interface.
// This replaces the old centralized labels.GetLabels() function.
func GetLabelsFromRecord(record *corev1.Record) []labels.Label {
	adapter := adapters.NewRecordAdapter(record)

	recordData, err := adapter.GetRecordData()
	if err != nil {
		return nil
	}

	if provider, ok := recordData.(types.LabelProvider); ok {
		return provider.GetAllLabels()
	}

	return nil
}
