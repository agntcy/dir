// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package gorm

import (
	"strings"

	"github.com/agntcy/dir/server/database/utils"
	"github.com/agntcy/dir/server/types"
	"gorm.io/gorm"
)

// applyExclusionFilters adds NOT-matching conditions for every populated field
// on ex. Joined/multi-valued fields (skills, domains, modules, locators,
// annotations) compile to a correlated NOT EXISTS subquery so a record that
// matches an excluded value via one row (e.g. skill "python") is not
// resurrected by another row (e.g. skill "nlp") — negating the include-path
// JOIN predicate directly would produce exactly that bug. Scalar records
// columns compile to a NULL-safe NOT(...).
//
// Split into scalar/join halves purely to keep cyclomatic complexity within
// lint limits; both are always applied together.
func applyExclusionFilters(query *gorm.DB, ex *types.ExcludedRecordFilters) *gorm.DB {
	query = applyExcludedScalarFilters(query, ex)
	query = applyExcludedJoinFilters(query, ex)

	return query
}

// applyExcludedScalarFilters handles the single-column records fields.
func applyExcludedScalarFilters(query *gorm.DB, ex *types.ExcludedRecordFilters) *gorm.DB {
	if len(ex.Names) > 0 {
		inner, args := utils.BuildWildcardCondition("records.name", ex.Names)
		query = query.Where(utils.BuildNegatedCondition("records.name", inner, false), args...)
	}

	if len(ex.Versions) > 0 {
		inner, args := utils.BuildComparisonConditions("records.version", ex.Versions)
		query = query.Where(utils.BuildNegatedCondition("records.version", inner, false), args...)
	}

	if len(ex.SchemaVersions) > 0 {
		inner, args := utils.BuildComparisonConditions("records.schema_version", ex.SchemaVersions)
		query = query.Where(utils.BuildNegatedCondition("records.schema_version", inner, true), args...)
	}

	if len(ex.Descriptions) > 0 {
		inner, args := utils.BuildWildcardCondition("records.description", ex.Descriptions)
		query = query.Where(utils.BuildNegatedCondition("records.description", inner, true), args...)
	}

	if len(ex.CreatedAts) > 0 {
		inner, args := utils.BuildComparisonConditions("records.oasf_created_at", ex.CreatedAts)
		// nullable=true: oasf_created_at has no NOT NULL constraint. AddRecord always
		// writes '' today so this isn't reachable yet, but a future migration on an
		// already-populated table could leave real NULLs here — guard against the
		// same NOT(NULL)-drops-the-row bug class the authors/description columns hit.
		query = query.Where(utils.BuildNegatedCondition("records.oasf_created_at", inner, true), args...)
	}

	if len(ex.Authors) > 0 {
		query = applyExcludedAuthors(query, ex.Authors)
	}

	return query
}

// applyExcludedJoinFilters handles the joined/multi-valued records fields.
func applyExcludedJoinFilters(query *gorm.DB, ex *types.ExcludedRecordFilters) *gorm.DB {
	if len(ex.SkillNames) > 0 || len(ex.SkillIDs) > 0 {
		query = applyExcludedSkills(query, ex.SkillNames, ex.SkillIDs)
	}

	if len(ex.DomainNames) > 0 || len(ex.DomainIDs) > 0 {
		query = applyExcludedDomains(query, ex.DomainNames, ex.DomainIDs)
	}

	if len(ex.ModuleNames) > 0 || len(ex.ModuleIDs) > 0 {
		query = applyExcludedModules(query, ex.ModuleNames, ex.ModuleIDs)
	}

	if len(ex.LocatorTypes) > 0 || len(ex.LocatorURLs) > 0 {
		query = applyExcludedLocators(query, ex.LocatorTypes, ex.LocatorURLs)
	}

	if len(ex.AnnotationKeys) > 0 || len(ex.AnnotationValues) > 0 {
		query = applyExcludedAnnotations(query, ex.AnnotationKeys, ex.AnnotationValues)
	}

	if len(ex.ScanSeverities) > 0 {
		query = applyExcludedScanSeverities(query, ex.ScanSeverities)
	}

	if len(ex.Owners) > 0 {
		query = applyExcludedOwners(query, ex.Owners)
	}

	return query
}

// applyExcludedAuthors excludes records whose authors JSON array contains an
// entry matching any pattern. Uses the same per-pattern wildcard-wrapping as
// the include path (records.authors is matched as "*pattern*").
func applyExcludedAuthors(query *gorm.DB, authors []string) *gorm.DB {
	var conditions []string

	var args []any

	for _, author := range authors {
		condition, arg := utils.BuildSingleWildcardCondition("records.authors", "*"+author+"*")
		conditions = append(conditions, condition)
		args = append(args, arg)
	}

	inner := conditions[0]
	if len(conditions) > 1 {
		inner = "(" + strings.Join(conditions, " OR ") + ")"
	}

	// nullable=true: records.authors has no NOT NULL constraint, and gorm's JSON
	// serializer writes a genuine SQL NULL (not an empty-array literal) for a nil
	// Go slice. Without this, NOT(NULL) evaluates to NULL and silently drops every
	// author-less record from the result set instead of retaining it.
	return query.Where(utils.BuildNegatedCondition("records.authors", inner, true), args...)
}

// applyExcludedSkills excludes records that have any skill matching the given
// names or IDs, via NOT EXISTS so a record with skills [nlp, python] excluded
// on "nlp" is still excluded (not resurrected by the python row).
//
// ID and name conditions are OR'd, not AND'd: unlike locator's type:url or
// annotation's key:value (one combined value from a single query, correctly
// AND'd — see applyExcludedLocators), SkillIDs and SkillNames arrive from
// independent RecordQuery entries with no pairing between them. Excluding
// "skill ID 5" and "skill name nlp" must exclude a record matching either —
// even via two different skill rows — not only a record with one row that
// happens to satisfy both simultaneously.
func applyExcludedSkills(query *gorm.DB, names []string, ids []uint64) *gorm.DB {
	var matchConditions []string

	var args []any

	if len(ids) > 0 {
		matchConditions = append(matchConditions, "ex.skill_id IN ?")
		args = append(args, ids)
	}

	if len(names) > 0 {
		nameCond, nameArgs := utils.BuildWildcardCondition("ex.name", names)
		matchConditions = append(matchConditions, nameCond)
		args = append(args, nameArgs...)
	}

	inner := "ex.record_cid = records.record_cid AND (" + strings.Join(matchConditions, " OR ") + ")"

	return query.Where(utils.BuildNotExistsCondition("skills", "ex", inner), args...)
}

// applyExcludedDomains mirrors applyExcludedSkills for the domains table —
// see applyExcludedSkills for why ID and name are OR'd, not AND'd.
func applyExcludedDomains(query *gorm.DB, names []string, ids []uint64) *gorm.DB {
	var matchConditions []string

	var args []any

	if len(ids) > 0 {
		matchConditions = append(matchConditions, "ex.domain_id IN ?")
		args = append(args, ids)
	}

	if len(names) > 0 {
		nameCond, nameArgs := utils.BuildWildcardCondition("ex.name", names)
		matchConditions = append(matchConditions, nameCond)
		args = append(args, nameArgs...)
	}

	inner := "ex.record_cid = records.record_cid AND (" + strings.Join(matchConditions, " OR ") + ")"

	return query.Where(utils.BuildNotExistsCondition("domains", "ex", inner), args...)
}

// applyExcludedModules mirrors applyExcludedSkills for the modules table —
// see applyExcludedSkills for why ID and name are OR'd, not AND'd.
func applyExcludedModules(query *gorm.DB, names []string, ids []uint64) *gorm.DB {
	var matchConditions []string

	var args []any

	if len(ids) > 0 {
		matchConditions = append(matchConditions, "ex.module_id IN ?")
		args = append(args, ids)
	}

	if len(names) > 0 {
		nameCond, nameArgs := utils.BuildWildcardCondition("ex.name", names)
		matchConditions = append(matchConditions, nameCond)
		args = append(args, nameArgs...)
	}

	inner := "ex.record_cid = records.record_cid AND (" + strings.Join(matchConditions, " OR ") + ")"

	return query.Where(utils.BuildNotExistsCondition("modules", "ex", inner), args...)
}

// applyExcludedLocators excludes records with a matching locator row. Known
// limitation (documented, not fixed — matches a pre-existing conflation in
// the include path): a negated LOCATOR query of the form "type:url" ANDs both
// conditions into one NOT EXISTS, so multiple negated locators cross-product
// across type/URL pairs rather than pairing per-query. Fixing this requires
// restructuring locator filters into (type, url) pairs, out of scope here.
func applyExcludedLocators(query *gorm.DB, locatorTypes []string, urls []string) *gorm.DB {
	var conditions []string

	var args []any

	conditions = append(conditions, "ex.record_cid = records.record_cid")

	if len(locatorTypes) > 0 {
		typeCond, typeArgs := utils.BuildWildcardCondition("ex.type", locatorTypes)
		conditions = append(conditions, typeCond)
		args = append(args, typeArgs...)
	}

	if len(urls) > 0 {
		urlCond, urlArgs := utils.BuildWildcardCondition("ex.url", urls)
		conditions = append(conditions, urlCond)
		args = append(args, urlArgs...)
	}

	inner := strings.Join(conditions, " AND ")

	return query.Where(utils.BuildNotExistsCondition("locators", "ex", inner), args...)
}

// applyExcludedAnnotations mirrors applyExcludedLocators for the annotations
// table; same key/value conflation limitation applies (see applyExcludedLocators).
func applyExcludedAnnotations(query *gorm.DB, keys []string, values []string) *gorm.DB {
	var conditions []string

	var args []any

	conditions = append(conditions, "ex.record_cid = records.record_cid")

	if len(keys) > 0 {
		keyCond, keyArgs := utils.BuildWildcardCondition("ex.key", keys)
		conditions = append(conditions, keyCond)
		args = append(args, keyArgs...)
	}

	if len(values) > 0 {
		valueCond, valueArgs := utils.BuildWildcardCondition("ex.value", values)
		conditions = append(conditions, valueCond)
		args = append(args, valueArgs...)
	}

	inner := strings.Join(conditions, " AND ")

	return query.Where(utils.BuildNotExistsCondition("annotations", "ex", inner), args...)
}

// applyExcludedScanSeverities excludes records that have any scan_reports row
// at or above any of the given thresholds — "nothing at or above this severity".
// applyExcludedOwners excludes records that have an ownership claim matching
// any of the given owner ID patterns.
func applyExcludedOwners(query *gorm.DB, owners []string) *gorm.DB {
	inner, args := utils.BuildWildcardCondition("ex.owner_id", owners)

	return query.Where(utils.BuildNotExistsCondition("owners", "ex", "ex.record_cid = records.record_cid AND ("+inner+")"), args...)
}

func applyExcludedScanSeverities(query *gorm.DB, thresholds []string) *gorm.DB {
	var severities []string
	for _, threshold := range thresholds {
		severities = append(severities, scanSeveritiesGTE(threshold)...)
	}

	if len(severities) == 0 {
		return query
	}

	return query.Where(
		utils.BuildNotExistsCondition("scan_reports", "ex", "ex.record_cid = records.record_cid AND ex.max_severity IN ?"),
		severities,
	)
}
