// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package v1

import (
	"errors"
	"fmt"
	"sort"

	coretypes "github.com/agntcy/dir/api/core/types"
	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/oasf-sdk/pkg/decoder"
	"google.golang.org/protobuf/types/known/structpb"
)

const (
	// CatalogMediaType is the media type used for a container
	// entry carrying an embedded or nested AI Catalog.
	CatalogMediaType = "application/ai-catalog+json"

	// CatalogHostURN is the authority segment used in entry URNs
	// ("urn:ai:<host>:cid:<cid>[:<suffix>]").
	CatalogHostURN = "org.agntcy"

	// CatalogSpecVersion is the AI Catalog spec version.
	CatalogSpecVersion = "1.0"

	// Protocol-specific media types supported by AI Catalog.
	ProtocolA2ACardJsonMediaType   = "application/a2a-agent-card+json"
	ProtocolMCPCardJsonMediaType   = "application/mcp-server-card+json"
	ProtocolAgentSkillsMdMediaType = "application/agentskill+md"
)

// catalogModuleProjection captures the per-module projection rules: the
// media type a module advertises, the URN suffix used to disambiguate the
// module's entry inside a multi-module container, and a short human label
// used to synthesise nested display names.
type catalogModuleProjection struct {
	MediaType string
	URNSuffix string
	Label     string
}

// catalogModules maps OASF integration module names onto their AI Catalog
// projection rules. A record is projectable only if it carries at least
// one of these modules.
//
// Mapped from OASF module names: https://schema.oasf.outshift.com/modules
var catalogModules = map[string]catalogModuleProjection{
	"integration/mcp": {
		MediaType: ProtocolMCPCardJsonMediaType, URNSuffix: "mcp", Label: "MCP",
	},
	"integration/a2a": {
		MediaType: ProtocolA2ACardJsonMediaType, URNSuffix: "a2a", Label: "A2A",
	},
	"core/language_model/agentskills": {
		MediaType: ProtocolAgentSkillsMdMediaType, URNSuffix: "agentskill", Label: "Skill",
	},
}

// RecordToCatalog projects an OASF record onto its AI Catalog entry
// representation, returned as a structpb.Struct. The result depends
// on how many known integration modules the record carries.
//
//   - 0 known modules → error; a catalog entry MUST point at an artifact
//     and there is nothing to project.
//   - 1 known module  → a LEAF entry whose media_type matches the module
//     (e.g. "application/a2a-agent-card+json") and whose `data` is the
//     module's structured data.
//   - 2+ known modules → a CONTAINER entry with media_type
//     "application/ai-catalog+json" whose `data` embeds a nested AI Catalog
//     with one entry per known module.
//
// The projection is deliberately pure: trust/identity metadata (e.g.
// signature-derived TrustManifests) and any publisher/host data beyond the
// URN host are layered on by the caller and intentionally not produced
// here.
func RecordToCatalog(record *corev1.Record) (*CatalogEntry, error) {
	if record == nil {
		return nil, errors.New("record is nil")
	}

	// Get adapter
	adapter, err := record.Decode()
	if err != nil {
		return nil, fmt.Errorf("failed to get record adapter: %w", err)
	}

	// Extract valid modules
	modules := knownCatalogModules(adapter)
	if len(modules) == 0 {
		return nil, errors.New("record has no known catalog modules")
	}

	// Single known module — leaf entry on the parent URN.
	if len(modules) == 1 {
		entry := moduleToCatalogEntry(modules[0])
		if entry == nil {
			return nil, errors.New("failed to project module to catalog entry")
		}

		return &CatalogEntry{
			Identifier:  catalogURN(record.GetCid(), ""),
			DisplayName: adapter.GetName(),
			Version:     new(adapter.GetVersion()),
			Description: new(adapter.GetDescription()),
			UpdatedAt:   new(adapter.GetCreatedAt()),
			MediaType:   entry.GetMediaType(),
			Artifact:    entry.GetArtifact(),
			Tags:        catalogTags(adapter),
		}, nil
	}

	// Multiple known modules — container entry on the parent URN, with one
	// nested entry per module.
	parentCID := record.GetCid()
	parentName := adapter.GetName()

	entries := make([]*CatalogEntry, 0, len(modules))
	for _, module := range modules {
		entry := moduleToCatalogEntry(module)
		if entry == nil {
			return nil, fmt.Errorf("failed to project module %q to catalog entry", module.GetName())
		}

		entries = append(entries, &CatalogEntry{
			Identifier:  catalogURN(parentCID, catalogModules[module.GetName()].URNSuffix),
			DisplayName: fmt.Sprintf("%s - %s", parentName, catalogModules[module.GetName()].Label),
			MediaType:   entry.GetMediaType(),
			Artifact:    entry.GetArtifact(),
		})
	}

	// Sort entries by URN suffix for deterministic output
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].GetIdentifier() < entries[j].GetIdentifier()
	})

	// Create container entry with nested catalog data
	container, err := decoder.StructToProto(&AICatalog{
		SpecVersion: CatalogSpecVersion,
		Entries:     entries,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to convert container to proto: %w", err)
	}

	return &CatalogEntry{
		Identifier:  catalogURN(parentCID, ""),
		DisplayName: parentName,
		Description: new(adapter.GetDescription()),
		Version:     new(adapter.GetVersion()),
		UpdatedAt:   new(adapter.GetCreatedAt()),
		MediaType:   CatalogMediaType,
		Tags:        catalogTags(adapter),
		Artifact: &CatalogEntry_Data{
			Data: structpb.NewStructValue(container),
		},
	}, nil
}

// moduleToCatalogEntry builds a leaf catalog entry for a single module.
//
// A catalog entry requires exactly one of `url` or `data`. We always carry
// the module's structured data inline via `data` in OASF.
func moduleToCatalogEntry(module coretypes.Module) *CatalogEntry {
	proj, known := catalogModules[module.GetName()]
	if !known {
		return nil
	}

	data, err := structpb.NewStruct(module.GetData())
	if err != nil {
		return nil
	}

	return &CatalogEntry{
		MediaType: proj.MediaType,
		Artifact: &CatalogEntry_Data{
			Data: structpb.NewStructValue(data),
		},
	}
}

// knownCatalogModules returns the record's modules that have a catalog
// projection rule, sorted by name for deterministic output.
func knownCatalogModules(rd coretypes.Record) []coretypes.Module {
	modules := rd.GetModules()
	out := make([]coretypes.Module, 0, len(modules))

	for _, module := range modules {
		// Skip modules with no known projection rule
		if _, known := catalogModules[module.GetName()]; !known {
			continue
		}

		// Skip modules with no data
		if module.GetData() == nil {
			continue
		}

		out = append(out, module)
	}

	sort.Slice(out, func(i, j int) bool { return out[i].GetName() < out[j].GetName() })

	return out
}

// catalogTags assembles the OASF-taxonomy + annotation tag list shared by
// leaf and container entries: one tag per skill and domain
// ("oasf:v<schema>:skills:<name>" / "oasf:v<schema>:domain:<name>"),
// followed by record annotations ("key" or "key=value").
func catalogTags(rd coretypes.Record) []string {
	out := make([]string, 0)

	for _, skill := range rd.GetSkills() {
		out = append(out, fmt.Sprintf("oasf:%s:skills:%s", rd.GetSchemaVersion(), skill.GetName()))
	}

	for _, domain := range rd.GetDomains() {
		out = append(out, fmt.Sprintf("oasf:%s:domains:%s", rd.GetSchemaVersion(), domain.GetName()))
	}

	// Sort output before appending annotation tags
	sort.Strings(out)

	// Append annotations as tags
	for key, value := range rd.GetAnnotations() {
		if value == "" {
			out = append(out, key)
		} else {
			out = append(out, fmt.Sprintf("%s=%s", key, value))
		}
	}

	return out
}

// catalogURN builds "urn:ai:<host>:cid:<cid>" with an optional ":<suffix>"
// used to disambiguate nested entries inside a container.
func catalogURN(cid, suffix string) string {
	base := fmt.Sprintf("urn:ai:%s:cid:%s", CatalogHostURN, cid)
	if suffix == "" {
		return base
	}

	return base + ":" + suffix
}
