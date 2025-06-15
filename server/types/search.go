// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package types

type SkillObject interface {
	GetID() uint32
	GetName() string
}

type LocatorObject interface {
	GetType() string
	GetUrl() string
}

type ExtensionObject interface {
	GetName() string
	GetVersion() string
}

type RecordObject interface {
	GetName() string
	GetVersion() string

	GetSkillObjects() []SkillObject
	GetLocatorObjects() []LocatorObject
	GetExtensionObjects() []ExtensionObject
}

type Filters struct {
	Limit             int
	Offset            int
	Name              string
	Version           string
	SkillIDs          []uint32
	SkillNames        []string
	LocatorTypes      []string
	LocatorURLs       []string
	ExtensionNames    []string
	ExtensionVersions []string
}

type FilterOption func(*Filters)

// WithLimit sets the maximum number of records to return.
func WithLimit(limit int) FilterOption {
	return func(sc *Filters) {
		sc.Limit = limit
	}
}

// WithOffset sets pagination offset.
func WithOffset(offset int) FilterOption {
	return func(sc *Filters) {
		sc.Offset = offset
	}
}

// WithName Filters records by name (partial match).
func WithName(name string) FilterOption {
	return func(sc *Filters) {
		sc.Name = name
	}
}

// WithVersion Filters records by exact version.
func WithVersion(version string) FilterOption {
	return func(sc *Filters) {
		sc.Version = version
	}
}

// WithSkillIDs Filters records by skill IDs.
func WithSkillIDs(ids ...uint32) FilterOption {
	return func(sc *Filters) {
		sc.SkillIDs = ids
	}
}

// WithSkillNames Filters records by skill names.
func WithSkillNames(names ...string) FilterOption {
	return func(sc *Filters) {
		sc.SkillNames = names
	}
}

// WithLocatorTypes Filters records by locator types.
func WithLocatorTypes(types ...string) FilterOption {
	return func(sc *Filters) {
		sc.LocatorTypes = types
	}
}

// WithLocatorURLs Filters records by locator URLs.
func WithLocatorURLs(urls ...string) FilterOption {
	return func(sc *Filters) {
		sc.LocatorURLs = urls
	}
}

// WithExtensionNames Filters records by extension names.
func WithExtensionNames(names ...string) FilterOption {
	return func(sc *Filters) {
		sc.ExtensionNames = names
	}
}

// WithExtensionVersions Filters records by extension versions.
func WithExtensionVersions(versions ...string) FilterOption {
	return func(sc *Filters) {
		sc.ExtensionVersions = versions
	}
}

type SearchAPI interface {
	// AddRecord adds a new agent record to the search database.
	AddRecord(record RecordObject) error

	// GetRecords retrieves agent records based on the provided Filters.
	GetRecords(opts ...FilterOption) ([]RecordObject, error)
}
