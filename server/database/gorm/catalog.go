// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package gorm

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	catalogv1 "github.com/agntcy/dir/api/catalog/v1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
)

// ---------------------------------------------------------------------------
// AI Catalog projection
// ---------------------------------------------------------------------------
//
// These constants and the knownModules table centralise the OASF ↔ AI
// Catalog mapping used by ToCatalog. They live next to ToCatalog so the
// projection rules are easy to audit in one place.

const (
	// catalogURNHost is the authority segment baked into every catalog
	// entry URN: "urn:ai:<host>:cid:<record-cid>[:<suffix>]". Made a
	// constant so it can later be made configurable from HostInfo.
	catalogURNHost = "agntcy.org"

	// catalogSpecVersion is the AI Catalog spec version embedded in the
	// nested AICatalog produced for multi-module records.
	catalogSpecVersion = "1.0"

	// catalogContainerMediaType is the media_type of the container entry
	// produced when a record has more than one known module.
	catalogContainerMediaType = "application/ai-catalog+json"
)

// moduleProjection captures the per-module mapping rules.
type moduleProjection struct {
	// MediaType is the catalog entry's media_type value.
	MediaType string

	// URNSuffix is appended to the record-level URN for nested entries.
	URNSuffix string

	// Label is the short human-readable tag used when synthesising a
	// nested-entry display name from the parent record (e.g. "(MCP)").
	Label string
}

// knownModules maps OASF integration module names onto AI Catalog
// metadata.
var knownModules = map[string]moduleProjection{
	"integration/mcp":                 {MediaType: "application/mcp-server-card+json", URNSuffix: "mcp", Label: "MCP"},
	"integration/a2a":                 {MediaType: "application/a2a-agent-card+json", URNSuffix: "a2a", Label: "A2A"},
	"core/language_model/agentskills": {MediaType: "application/agentskill+zip", URNSuffix: "agentskill", Label: "Skill"},
}

func (r *Record) GetAgents(filters ...string) ([]any, error) {
	// Convert filters
	// = filters

	// Required filters to support:
	// - Appendix A: Filter Expression Syntax
	// - Filter by version, e.g. "version=1.0.0", or "include_versions=true" to include all versions
	// - No version filter, default to latest (last created_at)

	// Fetch
	// run a query here to fetch: records, join with skills, modules, domains, annotations
	// var records []Record
	// err := db.Preload("Skills").Preload("Modules").Preload("Domains").Preload("Annotations").Find(&records).Error
	// if err != nil {
	// 	return nil, err
	// }

	// // Apply conversion to catalog format. Signatures are NOT preloaded
	// // with the record (they live in their own table); fetch them per
	// // record so the trust manifest can be built without paying the cost
	// // for callers that don't need it.
	// result := make([]interface{}, len(records))
	// for i, record := range records {
	// 	var sigs []*SignatureVerification
	// 	if err := db.Where("record_cid = ?", record.RecordCID).Find(&sigs).Error; err != nil {
	// 		return nil, err
	// 	}
	// 	catalog, err := record.ToCatalog(sigs)
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// 	result[i] = catalog
	// }
	return nil, nil
}

// ToCatalog projects this database Record onto its AI Catalog
// representation. The shape depends on how many "known" OASF integration
// modules the record contains:
//
//   - 0 known modules → returns an error; a CatalogEntry MUST point at
//     an artifact (url or inline data) and we have nothing to point at.
//   - 1 known module  → returns a LEAF entry whose media_type matches
//     the module type (e.g. "application/a2a-agent-card+json") and
//     whose artifact is the module's URL or inline data.
//   - 2+ known modules → returns a CONTAINER entry whose media_type is
//     "application/ai-catalog+json" and whose `data` field embeds a
//     nested AICatalog with one CatalogEntry per known module.
//
// signatures carries SignatureVerification rows associated with the
// record.
//
//nolint:cyclop
func (r *Record) ToCatalog(signatures []*SignatureVerification) (*catalogv1.CatalogEntry, error) {
	if r == nil {
		return nil, fmt.Errorf("ToCatalog called on nil record")
	}

	modules := r.knownCatalogModules()
	if len(modules) == 0 {
		return nil, fmt.Errorf("record %q has no known catalog modules", r.RecordCID)
	}

	baseURN := r.catalogURN("")
	recordTags := r.catalogTags()
	updatedAt := r.catalogUpdatedAt()
	description := strings.TrimSpace(r.Description)
	trustManifest := buildTrustManifest(signatures)

	// Single known module — leaf entry on the parent URN.
	if len(modules) == 1 { //nolint:nestif
		m := modules[0]

		entry, err := r.moduleToCatalogEntry(m, baseURN)
		if err != nil {
			return nil, err
		}

		entry.DisplayName = firstNonEmpty(m.DisplayName, r.Name, baseURN)
		entry.Tags = recordTags

		if v := r.Version; v != "" {
			entry.Version = &v
		}

		if description != "" {
			entry.Description = &description
		}

		if updatedAt != "" {
			entry.UpdatedAt = &updatedAt
		}

		if trustManifest != nil {
			entry.TrustManifest = trustManifest
		}

		return entry, nil
	}

	// 2+ known modules — container whose data is a nested AICatalog.
	nestedEntries := make([]*catalogv1.CatalogEntry, 0, len(modules))

	for _, m := range modules {
		moduleURN := r.catalogURN(knownModules[m.Name].URNSuffix)

		entry, err := r.moduleToCatalogEntry(m, moduleURN)
		if err != nil {
			return nil, err
		}

		entry.DisplayName = r.moduleDisplayName(m)

		// Per-module tags would come from module-level annotations, which
		// the schema does not track yet. Record-level taxonomy lives on
		// the container so we deliberately leave nested tags empty.
		// TODO(ai-catalog): wire module-level annotations when added.

		nestedEntries = append(nestedEntries, entry)
	}

	nestedCatalog := &catalogv1.AICatalog{
		SpecVersion: catalogSpecVersion,
		Entries:     nestedEntries,
	}

	dataValue, err := aiCatalogToValue(nestedCatalog)
	if err != nil {
		return nil, fmt.Errorf("encode nested catalog: %w", err)
	}

	container := &catalogv1.CatalogEntry{
		Identifier:  baseURN,
		DisplayName: firstNonEmpty(r.Name, baseURN),
		MediaType:   catalogContainerMediaType,
		Artifact:    &catalogv1.CatalogEntry_Data{Data: dataValue},
		Tags:        recordTags,
	}

	if v := r.Version; v != "" {
		container.Version = &v
	}

	if description != "" {
		container.Description = &description
	}

	if updatedAt != "" {
		container.UpdatedAt = &updatedAt
	}

	if trustManifest != nil {
		container.TrustManifest = trustManifest
	}

	return container, nil
}

// moduleToCatalogEntry builds a single leaf CatalogEntry for a module.
// Identifier and artifact are set here; DisplayName and Tags are filled
// in by the caller because they differ between the single-module-leaf
// and multi-module-nested cases.
func (r *Record) moduleToCatalogEntry(m Module, identifier string) (*catalogv1.CatalogEntry, error) {
	proj, ok := knownModules[m.Name]
	if !ok {
		return nil, fmt.Errorf("module %q has no AI Catalog projection rule", m.Name)
	}

	entry := &catalogv1.CatalogEntry{
		Identifier: identifier,
		MediaType:  proj.MediaType,
	}

	// CatalogEntry.artifact is a required oneof of (url | data). Prefer a
	// real URL, fall back to inline data, and as last resort synthesise a
	// stable URN URL so the entry is still spec-valid for callers that
	// merely browse the catalog.
	switch {
	case strings.TrimSpace(m.ArtifactURL) != "":
		entry.Artifact = &catalogv1.CatalogEntry_Url{Url: m.ArtifactURL}

	case len(m.ArtifactData) > 0:
		val, err := mapToValue(m.ArtifactData)
		if err != nil {
			return nil, fmt.Errorf("module %q: encode artifact data: %w", m.Name, err)
		}

		entry.Artifact = &catalogv1.CatalogEntry_Data{Data: val}

	default:
		entry.Artifact = &catalogv1.CatalogEntry_Url{Url: identifier}
	}

	return entry, nil
}

// knownCatalogModules returns the subset of r.Modules that has an
// AI Catalog projection rule, sorted by name for deterministic output.
func (r *Record) knownCatalogModules() []Module {
	if len(r.Modules) == 0 {
		return nil
	}

	out := make([]Module, 0, len(r.Modules))

	for _, m := range r.Modules {
		if _, ok := knownModules[m.Name]; ok {
			out = append(out, m)
		}
	}

	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })

	return out
}

// catalogURN builds the record-scoped URN. When suffix is non-empty it is
// appended as ":<suffix>", which is used to disambiguate nested entries
// inside a multi-module container.
func (r *Record) catalogURN(suffix string) string {
	cid := r.RecordCID
	if cid == "" {
		// Records without a CID shouldn't reach this code under normal
		// flows, but during tests or partial fixtures we fall back to a
		// slugified name so we never emit an empty identifier (which the
		// spec rejects).
		cid = strings.ReplaceAll(strings.ToLower(strings.TrimSpace(r.Name)), " ", "-")
	}

	base := fmt.Sprintf("urn:ai:%s:cid:%s", catalogURNHost, cid)
	if suffix == "" {
		return base
	}

	return base + ":" + suffix
}

// moduleDisplayName picks the nested entry's human-readable name. The
// module's own DisplayName wins; otherwise we synthesise "<record name>
// (<module label>)" so each nested entry remains distinguishable.
func (r *Record) moduleDisplayName(m Module) string {
	if dn := strings.TrimSpace(m.DisplayName); dn != "" {
		return dn
	}

	label := knownModules[m.Name].Label

	switch {
	case r.Name != "" && label != "":
		return fmt.Sprintf("%s (%s)", r.Name, label)
	case r.Name != "":
		return r.Name
	case label != "":
		return label
	default:
		return m.Name
	}
}

// catalogTags assembles the OASF-taxonomy + annotation tag list shared by
// leaf and container entries.
func (r *Record) catalogTags() []string {
	schemaVer := strings.TrimPrefix(r.SchemaVersion, "v")
	if schemaVer == "" {
		schemaVer = "1"
	}

	out := make([]string, 0, len(r.Skills)+len(r.Domains)+len(r.Annotations))

	for _, s := range r.Skills {
		out = append(out, fmt.Sprintf("oasf:v%s:skills:%s", schemaVer, s.Name))
	}

	for _, d := range r.Domains {
		out = append(out, fmt.Sprintf("oasf:v%s:domain:%s", schemaVer, d.Name))
	}

	for _, a := range r.Annotations {
		if a.Value == "" {
			out = append(out, a.Key)
		} else {
			out = append(out, fmt.Sprintf("%s=%s", a.Key, a.Value))
		}
	}

	return out
}

// catalogUpdatedAt formats the record's UpdatedAt as RFC 3339 (the
// timestamp shape mandated by the AI Catalog spec). Empty on zero value.
func (r *Record) catalogUpdatedAt() string {
	if r.UpdatedAt.IsZero() {
		return ""
	}

	return r.UpdatedAt.UTC().Format(time.RFC3339)
}

// buildTrustManifest projects a set of signature verification rows onto
// an AI Catalog TrustManifest:
//
//   - identity      = primary verified signer's Identity()
//   - identityType  = primary verified signer's SignerType ("oidc", "key", ...)

// TODO(ai-catalog): surface attestations and a JCS-canonicalised
// detached-JWS signature on the manifest itself.
func buildTrustManifest(signatures []*SignatureVerification) *catalogv1.TrustManifest {
	verified := make([]*SignatureVerification, 0, len(signatures))

	for _, s := range signatures {
		if s == nil {
			continue
		}

		if s.Status == signatureStatusVerified {
			verified = append(verified, s)
		}
	}

	if len(verified) == 0 {
		return nil
	}

	sort.Slice(verified, func(i, j int) bool {
		if !verified[i].CreatedAt.Equal(verified[j].CreatedAt) {
			return verified[i].CreatedAt.Before(verified[j].CreatedAt)
		}

		return verified[i].SignerKey < verified[j].SignerKey
	})

	primary := verified[0]

	tm := &catalogv1.TrustManifest{
		Identity: primary.Identity(),
	}

	if it := strings.TrimSpace(primary.SignerType); it != "" {
		tm.IdentityType = &it
	}

	return tm
}

// firstNonEmpty returns the first non-empty string in values, or "" if
// none. Keeps the leaf/container fall-through chains readable.
func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}

	return ""
}

// mapToValue converts a JSON-shaped Go map (the form GORM scans JSON
// columns into) to a structpb.Value suitable for CatalogEntry.data.
func mapToValue(m map[string]any) (*structpb.Value, error) {
	if m == nil {
		return structpb.NewNullValue(), nil
	}

	return structpb.NewValue(m) //nolint:wrapcheck
}

// aiCatalogToValue encodes a typed AICatalog proto as a structpb.Value via
// a protojson round-trip. The spec mandates that CatalogEntry.data, when
// media_type is "application/ai-catalog+json", is itself a JSON-encoded
// AICatalog document — protojson handles the canonical JSON form for us.
func aiCatalogToValue(c *catalogv1.AICatalog) (*structpb.Value, error) {
	if c == nil {
		return structpb.NewNullValue(), nil
	}

	raw, err := protojson.Marshal(c)
	if err != nil {
		return nil, fmt.Errorf("marshal AICatalog: %w", err)
	}

	var generic any
	if err := json.Unmarshal(raw, &generic); err != nil {
		return nil, fmt.Errorf("unmarshal AICatalog JSON: %w", err)
	}

	return structpb.NewValue(generic) //nolint:wrapcheck
}

type CollectionSummary struct {
	Module  []string // can only be MCP, A2A, or Skills
	Skills  []string
	Domains []string
}

func (c *CollectionSummary) WellKnown() any {
	return `
{
	"specVersion": "1.0",
	"host": {
		"displayName": "Acme Enterprise AI",
		"identifier": "did:web:acme.com"
	},
	"entries": [], // published entries, can be MANY
	"collections": [
		{
			"displayName": "MCP Record Catalog",
			"url": "https://localhost:8080/agents?type=mcp",
			"description": "Returns all available MCP records."
		},
		{
			"displayName": "A2A Record Catalog",
			"url": "https://localhost:8080/agents?type=a2a",
			"description": "Returns all available A2A records."
		},
		{
			"displayName": "Agents with Skill X",
			"url": "https://localhost:8080/agents?type=skill_x&skill_name={skill_name}",
			"description": "Returns all available agents with Skill X."
		}
	]
}
  `
}
