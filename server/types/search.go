// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package types

type RecordFilters struct {
	Limit          int
	Offset         int
	Names          []string
	Versions       []string
	SkillIDs       []uint64
	SkillNames     []string
	LocatorTypes   []string
	LocatorURLs    []string
	ModuleNames    []string
	ModuleIDs      []uint64
	DomainIDs      []uint64
	DomainNames    []string
	CreatedAts     []string
	Authors        []string
	SchemaVersions []string
	Verified       *bool // Filter by verified status (name ownership verified via JWKS)
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
