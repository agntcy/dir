// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package types

// RegistryType represents the type of external registry to import from.
type RegistryType string

const (
	// RegistryTypeMCP represents the Model Context Protocol registry.
	RegistryTypeMCP RegistryType = "mcp"

	// FUTURE: RegistryTypeNANDA represents the NANDA registry.
	// RegistryTypeNANDA RegistryType = "nanda".

	// FUTURE:RegistryTypeA2A represents the Agent-to-Agent protocol registry.
	// RegistryTypeA2A RegistryType = "a2a".
)
