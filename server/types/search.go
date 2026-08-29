// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package types

type RecordFilters struct {
	Limit              int
	Offset             int
	CIDs               []string
	Names              []string
	Versions           []string
	SkillIDs           []uint64
	SkillNames         []string
	LocatorTypes       []string
	LocatorURLs        []string
	ModuleNames        []string
	ModuleIDs          []uint64
	DomainIDs          []uint64
	DomainNames        []string
	CreatedAts         []string
	Authors            []string
	SchemaVersions     []string
	Verified           *bool    // Filter by verified status (name ownership verified via JWKS)
	Trusted            *bool    // Filter by trusted status (signature verification passed)
	ScanSafe           *bool    // Filter by is_safe: true = all scanners safe, false = at least one unsafe
	ScanSeverities     []string // Filter by max scan severity >= threshold (e.g. "HIGH")
	ScanStatuses       []string // Filter by how completely a scan ran (e.g. "failed")
	ScanFailureReasons []string // Filter by why a scan produced no result (e.g. "source-unreachable")
	AnnotationKeys     []string
	AnnotationValues   []string
	Annotations        []Annotation
	Descriptions       []string              // Match against record description field.
	Owners             []string              // Filter by owner ID patterns (SPIFFE or similar identity).
	Excluded           ExcludedRecordFilters // Negated (exclude) counterparts of the fields above.

	OrderBy []RecordOrderClause // Order by directives applied in sequence.
}

// ExcludedRecordFilters mirrors the value-bearing fields of RecordFilters that
// support negation. Populated from RecordQuery entries with negate=true.
// Kept separate from RecordFilters rather than adding parallel Excluded*
// fields directly, since negated values combine with AND while included
// values combine with OR — the two cannot share one slice.
type ExcludedRecordFilters struct {
	Names              []string
	Versions           []string
	SchemaVersions     []string
	Descriptions       []string
	CreatedAts         []string
	Authors            []string
	SkillNames         []string
	SkillIDs           []uint64
	DomainNames        []string
	DomainIDs          []uint64
	ModuleNames        []string
	ModuleIDs          []uint64
	LocatorTypes       []string
	LocatorURLs        []string
	AnnotationKeys     []string
	AnnotationValues   []string
	ScanSeverities     []string
	ScanStatuses       []string
	ScanFailureReasons []string
	Owners             []string
}

type Annotation struct {
	Key   string
	Value string
}

// RecordOrderClause is a single ORDER BY directive.
type RecordOrderClause struct {
	Column string
	Desc   bool
}

type FilterOption func(*RecordFilters)

// WithLimit sets the maximum number of records to return.
func WithLimit(limit int) FilterOption {
	return func(sc *RecordFilters) {
		sc.Limit = limit
	}
}

// WithOffset sets pagination offset.
func WithOffset(offset int) FilterOption {
	return func(sc *RecordFilters) {
		sc.Offset = offset
	}
}

// WithCIDs filters records by content-addressable identifiers.
func WithCIDs(cids ...string) FilterOption {
	return func(sc *RecordFilters) {
		sc.CIDs = append(sc.CIDs, cids...)
	}
}

// WithNames filters records by name patterns.
func WithNames(names ...string) FilterOption {
	return func(sc *RecordFilters) {
		sc.Names = append(sc.Names, names...)
	}
}

// WithVersions filters records by version patterns.
func WithVersions(versions ...string) FilterOption {
	return func(sc *RecordFilters) {
		sc.Versions = append(sc.Versions, versions...)
	}
}

// WithSkillIDs RecordFilters records by skill IDs.
func WithSkillIDs(ids ...uint64) FilterOption {
	return func(sc *RecordFilters) {
		sc.SkillIDs = append(sc.SkillIDs, ids...)
	}
}

// WithSkillNames RecordFilters records by skill names.
func WithSkillNames(names ...string) FilterOption {
	return func(sc *RecordFilters) {
		sc.SkillNames = append(sc.SkillNames, names...)
	}
}

// WithLocatorTypes RecordFilters records by locator types.
func WithLocatorTypes(types ...string) FilterOption {
	return func(sc *RecordFilters) {
		sc.LocatorTypes = append(sc.LocatorTypes, types...)
	}
}

// WithLocatorURLs RecordFilters records by locator URLs.
func WithLocatorURLs(urls ...string) FilterOption {
	return func(sc *RecordFilters) {
		sc.LocatorURLs = append(sc.LocatorURLs, urls...)
	}
}

// WithModuleNames RecordFilters records by module names.
func WithModuleNames(names ...string) FilterOption {
	return func(sc *RecordFilters) {
		sc.ModuleNames = append(sc.ModuleNames, names...)
	}
}

// WithDomainIDs filters records by domain IDs.
func WithDomainIDs(ids ...uint64) FilterOption {
	return func(sc *RecordFilters) {
		sc.DomainIDs = append(sc.DomainIDs, ids...)
	}
}

// WithDomainNames filters records by domain names.
func WithDomainNames(names ...string) FilterOption {
	return func(sc *RecordFilters) {
		sc.DomainNames = append(sc.DomainNames, names...)
	}
}

// WithCreatedAts filters records by created_at timestamp patterns.
func WithCreatedAts(createdAts ...string) FilterOption {
	return func(sc *RecordFilters) {
		sc.CreatedAts = append(sc.CreatedAts, createdAts...)
	}
}

// WithOrderBy appends ORDER BY directives.
func WithOrderBy(clauses ...RecordOrderClause) FilterOption {
	return func(sc *RecordFilters) {
		sc.OrderBy = append(sc.OrderBy, clauses...)
	}
}

// WithAuthors filters records by author names.
func WithAuthors(names ...string) FilterOption {
	return func(sc *RecordFilters) {
		sc.Authors = append(sc.Authors, names...)
	}
}

// WithSchemaVersions filters records by schema version patterns.
func WithSchemaVersions(versions ...string) FilterOption {
	return func(sc *RecordFilters) {
		sc.SchemaVersions = append(sc.SchemaVersions, versions...)
	}
}

// WithModuleIDs filters records by module IDs.
func WithModuleIDs(ids ...uint64) FilterOption {
	return func(sc *RecordFilters) {
		sc.ModuleIDs = append(sc.ModuleIDs, ids...)
	}
}

// WithVerified filters records by verified status.
func WithVerified(verified bool) FilterOption {
	return func(sc *RecordFilters) {
		sc.Verified = &verified
	}
}

// WithTrusted filters records by trusted status (signature verification passed).
func WithTrusted(trusted bool) FilterOption {
	return func(sc *RecordFilters) {
		sc.Trusted = &trusted
	}
}

// WithScanSafe filters records by whether all scanners reported is_safe = true.
func WithScanSafe(safe bool) FilterOption {
	return func(sc *RecordFilters) {
		sc.ScanSafe = &safe
	}
}

// WithScanSeverities filters records whose highest scan severity meets or exceeds a threshold.
func WithScanSeverities(severities ...string) FilterOption {
	return func(sc *RecordFilters) {
		sc.ScanSeverities = append(sc.ScanSeverities, severities...)
	}
}

// WithAnnotationKeys filters records by annotation key patterns.
func WithAnnotationKeys(keys ...string) FilterOption {
	return func(sc *RecordFilters) {
		sc.AnnotationKeys = append(sc.AnnotationKeys, keys...)
	}
}

// WithAnnotationValues filters records by annotation value patterns.
func WithAnnotationValues(values ...string) FilterOption {
	return func(sc *RecordFilters) {
		sc.AnnotationValues = append(sc.AnnotationValues, values...)
	}
}

func WithAnnotations(annotations ...Annotation) FilterOption {
	return func(sc *RecordFilters) {
		sc.Annotations = append(sc.Annotations, annotations...)
	}
}

// WithDescriptions filters records by description patterns.
func WithDescriptions(descriptions ...string) FilterOption {
	return func(sc *RecordFilters) {
		sc.Descriptions = append(sc.Descriptions, descriptions...)
	}
}

// WithOwners filters records by owner ID patterns.
func WithOwners(owners ...string) FilterOption {
	return func(sc *RecordFilters) {
		sc.Owners = append(sc.Owners, owners...)
	}
}

// WithoutOwners excludes records whose owner ID matches any of the given patterns.
func WithoutOwners(owners ...string) FilterOption {
	return func(sc *RecordFilters) {
		sc.Excluded.Owners = append(sc.Excluded.Owners, owners...)
	}
}

// WithoutNames excludes records whose name matches any of the given patterns.
func WithoutNames(names ...string) FilterOption {
	return func(sc *RecordFilters) {
		sc.Excluded.Names = append(sc.Excluded.Names, names...)
	}
}

// WithoutVersions excludes records whose version matches any of the given patterns.
func WithoutVersions(versions ...string) FilterOption {
	return func(sc *RecordFilters) {
		sc.Excluded.Versions = append(sc.Excluded.Versions, versions...)
	}
}

// WithoutSchemaVersions excludes records whose schema version matches any of the given patterns.
func WithoutSchemaVersions(versions ...string) FilterOption {
	return func(sc *RecordFilters) {
		sc.Excluded.SchemaVersions = append(sc.Excluded.SchemaVersions, versions...)
	}
}

// WithoutDescriptions excludes records whose description matches any of the given patterns.
func WithoutDescriptions(descriptions ...string) FilterOption {
	return func(sc *RecordFilters) {
		sc.Excluded.Descriptions = append(sc.Excluded.Descriptions, descriptions...)
	}
}

// WithoutCreatedAts excludes records whose created_at matches any of the given patterns.
func WithoutCreatedAts(createdAts ...string) FilterOption {
	return func(sc *RecordFilters) {
		sc.Excluded.CreatedAts = append(sc.Excluded.CreatedAts, createdAts...)
	}
}

// WithoutAuthors excludes records whose authors match any of the given patterns.
func WithoutAuthors(names ...string) FilterOption {
	return func(sc *RecordFilters) {
		sc.Excluded.Authors = append(sc.Excluded.Authors, names...)
	}
}

// WithoutSkillNames excludes records that have a skill matching any of the given patterns.
func WithoutSkillNames(names ...string) FilterOption {
	return func(sc *RecordFilters) {
		sc.Excluded.SkillNames = append(sc.Excluded.SkillNames, names...)
	}
}

// WithoutSkillIDs excludes records that have a skill with any of the given IDs.
func WithoutSkillIDs(ids ...uint64) FilterOption {
	return func(sc *RecordFilters) {
		sc.Excluded.SkillIDs = append(sc.Excluded.SkillIDs, ids...)
	}
}

// WithoutDomainNames excludes records that have a domain matching any of the given patterns.
func WithoutDomainNames(names ...string) FilterOption {
	return func(sc *RecordFilters) {
		sc.Excluded.DomainNames = append(sc.Excluded.DomainNames, names...)
	}
}

// WithoutDomainIDs excludes records that have a domain with any of the given IDs.
func WithoutDomainIDs(ids ...uint64) FilterOption {
	return func(sc *RecordFilters) {
		sc.Excluded.DomainIDs = append(sc.Excluded.DomainIDs, ids...)
	}
}

// WithoutModuleNames excludes records that have a module matching any of the given patterns.
func WithoutModuleNames(names ...string) FilterOption {
	return func(sc *RecordFilters) {
		sc.Excluded.ModuleNames = append(sc.Excluded.ModuleNames, names...)
	}
}

// WithoutModuleIDs excludes records that have a module with any of the given IDs.
func WithoutModuleIDs(ids ...uint64) FilterOption {
	return func(sc *RecordFilters) {
		sc.Excluded.ModuleIDs = append(sc.Excluded.ModuleIDs, ids...)
	}
}

// WithoutLocatorTypes excludes records that have a locator type matching any of the given patterns.
func WithoutLocatorTypes(types ...string) FilterOption {
	return func(sc *RecordFilters) {
		sc.Excluded.LocatorTypes = append(sc.Excluded.LocatorTypes, types...)
	}
}

// WithoutLocatorURLs excludes records that have a locator URL matching any of the given patterns.
func WithoutLocatorURLs(urls ...string) FilterOption {
	return func(sc *RecordFilters) {
		sc.Excluded.LocatorURLs = append(sc.Excluded.LocatorURLs, urls...)
	}
}

// WithoutAnnotationKeys excludes records that have an annotation key matching any of the given patterns.
func WithoutAnnotationKeys(keys ...string) FilterOption {
	return func(sc *RecordFilters) {
		sc.Excluded.AnnotationKeys = append(sc.Excluded.AnnotationKeys, keys...)
	}
}

// WithoutAnnotationValues excludes records that have an annotation value matching any of the given patterns.
func WithoutAnnotationValues(values ...string) FilterOption {
	return func(sc *RecordFilters) {
		sc.Excluded.AnnotationValues = append(sc.Excluded.AnnotationValues, values...)
	}
}

// WithoutScanSeverities excludes records whose highest scan severity meets or exceeds any of the given thresholds.
func WithoutScanSeverities(severities ...string) FilterOption {
	return func(sc *RecordFilters) {
		sc.Excluded.ScanSeverities = append(sc.Excluded.ScanSeverities, severities...)
	}
}

// WithScanStatuses filters records that have a scan result with any of the
// given statuses ("completed", "partial", "failed").
func WithScanStatuses(statuses ...string) FilterOption {
	return func(sc *RecordFilters) {
		sc.ScanStatuses = append(sc.ScanStatuses, statuses...)
	}
}

// WithoutScanStatuses excludes records that have a scan result with any of the
// given statuses.
func WithoutScanStatuses(statuses ...string) FilterOption {
	return func(sc *RecordFilters) {
		sc.Excluded.ScanStatuses = append(sc.Excluded.ScanStatuses, statuses...)
	}
}

// WithScanFailureReasons filters records that have a scan result whose failure
// reason matches any of the given values.
func WithScanFailureReasons(reasons ...string) FilterOption {
	return func(sc *RecordFilters) {
		sc.ScanFailureReasons = append(sc.ScanFailureReasons, reasons...)
	}
}

// WithoutScanFailureReasons excludes records that have a scan result whose
// failure reason matches any of the given values.
func WithoutScanFailureReasons(reasons ...string) FilterOption {
	return func(sc *RecordFilters) {
		sc.Excluded.ScanFailureReasons = append(sc.Excluded.ScanFailureReasons, reasons...)
	}
}
