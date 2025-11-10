// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package types

type RecordFilters struct {
	Limit        int
	Offset       int
	Name         string
	Version      string
	SkillIDs     []uint64
	SkillNames   []string
	LocatorTypes []string
	LocatorURLs  []string
	ModuleNames  []string
	DomainIDs    []uint64
	DomainNames  []string
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

// WithName RecordFilters records by name (partial match).
func WithName(name string) FilterOption {
	return func(sc *RecordFilters) {
		sc.Name = name
	}
}

// WithVersion RecordFilters records by exact version.
func WithVersion(version string) FilterOption {
	return func(sc *RecordFilters) {
		sc.Version = version
	}
}

// WithSkillIDs RecordFilters records by skill IDs.
func WithSkillIDs(ids ...uint64) FilterOption {
	return func(sc *RecordFilters) {
		sc.SkillIDs = ids
	}
}

// WithSkillNames RecordFilters records by skill names.
func WithSkillNames(names ...string) FilterOption {
	return func(sc *RecordFilters) {
		sc.SkillNames = names
	}
}

// WithLocatorTypes RecordFilters records by locator types.
func WithLocatorTypes(types ...string) FilterOption {
	return func(sc *RecordFilters) {
		sc.LocatorTypes = types
	}
}

// WithLocatorURLs RecordFilters records by locator URLs.
func WithLocatorURLs(urls ...string) FilterOption {
	return func(sc *RecordFilters) {
		sc.LocatorURLs = urls
	}
}

// WithModuleNames RecordFilters records by module names.
func WithModuleNames(names ...string) FilterOption {
	return func(sc *RecordFilters) {
		sc.ModuleNames = names
	}
}

// WithDomainIDs filters records by domain IDs.
func WithDomainIDs(ids ...uint64) FilterOption {
	return func(sc *RecordFilters) {
		sc.DomainIDs = ids
	}
}

// WithDomainNames filters records by domain names.
func WithDomainNames(names ...string) FilterOption {
	return func(sc *RecordFilters) {
		sc.DomainNames = names
	}
}
