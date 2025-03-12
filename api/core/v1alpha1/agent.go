// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package corev1alpha1

func removeDuplicates[T comparable](slice []T) []T {
	keys := make(map[T]struct{})
	result := make([]T, 0, len(slice))

	for _, item := range slice {
		if _, exists := keys[item]; !exists {
			keys[item] = struct{}{}

			result = append(result, item)
		}
	}

	return result
}

//nolint:gocognit,cyclop
func (x *Agent) Merge(other *Agent) {
	if x == nil {
		return
	}

	// Merge annotations, keeping receiver's values when keys conflict
	if x.GetAnnotations() == nil {
		x.Annotations = make(map[string]string)
	}

	if other == nil {
		return
	}

	// Only use other's scalar fields if receiver doesn't have them set
	if x.GetName() == "" {
		x.Name = other.GetName()
	}

	if x.GetVersion() == "" {
		x.Version = other.GetVersion()
	}

	if x.GetCreatedAt() == nil {
		x.CreatedAt = other.GetCreatedAt()
	}

	if x.GetDigest() == nil {
		x.Digest = other.GetDigest()
	}

	// Merge slices without duplicates, keeping receiver's values first
	if len(other.GetAuthors()) > 0 {
		x.Authors = removeDuplicates(append(other.GetAuthors(), x.GetAuthors()...))
	}

	if len(other.GetSkills()) > 0 {
		x.Skills = removeDuplicates(append(other.GetSkills(), x.GetSkills()...))
	}

	for k, v := range other.GetAnnotations() {
		if _, exists := x.GetAnnotations()[k]; !exists {
			x.Annotations[k] = v
		}
	}

	// Merge Locators, keeping receiver's values when names conflict
	if len(other.GetLocators()) > 0 {
		locatorMap := make(map[string]*Locator)

		// Add other's locators first
		for _, loc := range other.GetLocators() {
			if loc != nil {
				locatorMap[loc.GetName()] = loc
			}
		}

		// Override with receiver's locators
		for _, loc := range x.GetLocators() {
			if loc != nil {
				locatorMap[loc.GetName()] = loc
			}
		}

		x.Locators = make([]*Locator, 0, len(locatorMap))
		for _, loc := range locatorMap {
			x.Locators = append(x.GetLocators(), loc)
		}
	}

	// Merge Extensions, keeping receiver's values when names conflict
	if len(other.GetExtensions()) > 0 {
		extensionMap := make(map[string]*Extension)

		// Add other's extensions first
		for _, ext := range other.GetExtensions() {
			if ext != nil {
				extensionMap[ext.GetName()] = ext
			}
		}

		// Override with receiver's extensions
		for _, ext := range x.GetExtensions() {
			if ext != nil {
				extensionMap[ext.GetName()] = ext
			}
		}

		x.Extensions = make([]*Extension, 0, len(extensionMap))
		for _, ext := range extensionMap {
			x.Extensions = append(x.Extensions, ext)
		}
	}
}
