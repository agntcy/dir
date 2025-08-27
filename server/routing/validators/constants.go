// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package validators

import "strings"

// NamespaceType defines the type for semantic label namespaces.
type NamespaceType string

const (
	// NamespaceSkills defines the namespace for skill-based labels.
	NamespaceSkills NamespaceType = "skills"

	// NamespaceDomains defines the namespace for domain-based labels.
	NamespaceDomains NamespaceType = "domains"

	// NamespaceFeatures defines the namespace for feature-based labels.
	NamespaceFeatures NamespaceType = "features"
)

// String returns the string representation of the namespace type.
func (n NamespaceType) String() string {
	return string(n)
}

// Example: NamespaceSkills.Prefix() returns "/skills/".
func (n NamespaceType) Prefix() string {
	return "/" + string(n) + "/"
}

// IsValid checks if the namespace type is one of the supported types.
func (n NamespaceType) IsValid() bool {
	switch n {
	case NamespaceSkills, NamespaceDomains, NamespaceFeatures:
		return true
	default:
		return false
	}
}

// ParseNamespace converts a string to NamespaceType if valid.
func ParseNamespace(s string) (NamespaceType, bool) {
	ns := NamespaceType(s)
	if ns.IsValid() {
		return ns, true
	}

	return "", false
}

// AllNamespaces returns all supported namespace types.
func AllNamespaces() []NamespaceType {
	return []NamespaceType{NamespaceSkills, NamespaceDomains, NamespaceFeatures}
}

// IsValidNamespaceKey checks if a key starts with any valid namespace prefix.
// Returns true if the key starts with /skills/, /domains/, or /features/.
func IsValidNamespaceKey(key string) bool {
	for _, ns := range AllNamespaces() {
		if strings.HasPrefix(key, ns.Prefix()) {
			return true
		}
	}

	return false
}

// GetNamespaceFromKey extracts the namespace type from a key.
// Returns the namespace type and true if found, or empty namespace and false if not found.
func GetNamespaceFromKey(key string) (NamespaceType, bool) {
	for _, ns := range AllNamespaces() {
		if strings.HasPrefix(key, ns.Prefix()) {
			return ns, true
		}
	}

	return "", false
}

const (
	// MinLabelKeyParts defines the minimum number of parts required in a label key after splitting.
	// Format: /type/label/CID splits into ["", "type", "label", "CID"] = 4 parts (empty first due to leading slash).
	MinLabelKeyParts = 4
)

const (
	DomainSchemaPrefix   = "schema.oasf.agntcy.org/domains/"
	FeaturesSchemaPrefix = "schema.oasf.agntcy.org/features/"
)
