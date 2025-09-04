// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package routing

import (
	"strings"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/server/routing/validators"
	"github.com/agntcy/dir/server/types/adapters"
	"github.com/agntcy/dir/utils/logging"
)

var labelLogger = logging.Logger("routing/labels")

// GetLabels extracts all labels from a record across all supported namespaces.
// This is a pure function that can be used by both local and remote routing operations.
//
// The function extracts labels from:
// - Skills: /skills/<skill_name>
// - Domains: /domains/<domain_name> (from extensions with domain schema prefix)
// - Features: /features/<feature_name> (from extensions with features schema prefix)
// - Locators: /locators/<locator_type>
//
// Returns a slice of fully qualified label strings with namespace prefixes.
func GetLabels(record *corev1.Record) []string {
	// Use adapter pattern to get version-agnostic access to record data
	adapter := adapters.NewRecordAdapter(record)

	recordData := adapter.GetRecordData()
	if recordData == nil {
		labelLogger.Error("failed to get record data")

		return nil
	}

	var labels []string

	// Extract record skills
	skills := make([]string, 0, len(recordData.GetSkills()))
	for _, skill := range recordData.GetSkills() {
		skills = append(skills, validators.NamespaceSkills.Prefix()+skill.GetName())
	}

	labels = append(labels, skills...)

	// Extract record domains from extensions
	var domains []string

	for _, ext := range recordData.GetExtensions() {
		if strings.HasPrefix(ext.GetName(), validators.DomainSchemaPrefix) {
			domain := ext.GetName()[len(validators.DomainSchemaPrefix):]
			domains = append(domains, validators.NamespaceDomains.Prefix()+domain)
		}
	}

	labels = append(labels, domains...)

	// Extract record features from extensions
	var features []string

	for _, ext := range recordData.GetExtensions() {
		if strings.HasPrefix(ext.GetName(), validators.FeaturesSchemaPrefix) {
			feature := ext.GetName()[len(validators.FeaturesSchemaPrefix):]
			features = append(features, validators.NamespaceFeatures.Prefix()+feature)
		}
	}

	labels = append(labels, features...)

	// Extract record locators
	locators := make([]string, 0, len(recordData.GetLocators()))
	for _, locator := range recordData.GetLocators() {
		locators = append(locators, validators.NamespaceLocators.Prefix()+locator.GetType())
	}

	labels = append(labels, locators...)

	return labels
}
