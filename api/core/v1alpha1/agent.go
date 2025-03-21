// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package corev1alpha1

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
)

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

func firstNonEmptyString(first, second string) string {
	if first != "" {
		return first
	}

	return second
}

func mergeItems[T any](receiverItems, otherItems []*T, getName func(*T) string) []*T {
	itemMap := make(map[string]*T)

	// Add other's items first
	for _, item := range otherItems {
		if item != nil {
			itemMap[getName(item)] = item
		}
	}

	// Override with receiver's items
	for _, item := range receiverItems {
		if item != nil {
			itemMap[getName(item)] = item
		}
	}

	mergedItems := make([]*T, 0, len(itemMap))
	for _, item := range itemMap {
		mergedItems = append(mergedItems, item)
	}

	return mergedItems
}

//nolint:gocognit,cyclop
func (a *Agent) Merge(other *Agent) {
	if other == nil {
		return
	}

	// Only use other's scalar fields if receiver doesn't have them set
	a.Name = firstNonEmptyString(a.GetName(), other.GetName())
	a.Version = firstNonEmptyString(a.GetVersion(), other.GetVersion())

	if a.GetCreatedAt() == "" {
		a.CreatedAt = other.GetCreatedAt()
	}

	// Merge slices without duplicates, keeping receiver's values first
	if len(other.GetAuthors()) > 0 {
		a.Authors = removeDuplicates(append(other.GetAuthors(), a.GetAuthors()...))
	}

	// Merge annotations, keeping receiver's values when keys conflict
	if a.GetAnnotations() == nil {
		a.Annotations = make(map[string]string)
	}

	for k, v := range other.GetAnnotations() {
		if _, exists := a.GetAnnotations()[k]; !exists {
			a.Annotations[k] = v
		}
	}

	// Merge Locators, keeping receiver's values when "type/url" conflict
	if len(other.GetLocators()) > 0 {
		a.Locators = mergeItems(
			a.GetLocators(),
			other.GetLocators(),
			func(locator *Locator) string {
				return path.Join(locator.GetType(), locator.GetUrl())
			},
		)
	}

	// Merge Extensions, keeping receiver's values when "name/version" conflict
	if len(other.GetExtensions()) > 0 {
		a.Extensions = mergeItems(
			a.GetExtensions(),
			other.GetExtensions(),
			func(extension *Extension) string {
				return path.Join(extension.GetName(), extension.GetVersion())
			},
		)
	}

	// Merge skills, keeping receiver's values when "key" conflict
	if len(other.GetSkills()) > 0 {
		a.Skills = mergeItems(
			a.GetSkills(),
			other.GetSkills(),
			func(skill *Skill) string {
				return skill.Key()
			},
		)
	}
}

func (a *Agent) LoadFromFile(path string) error {
	reader, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}

	data, err := io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("failed to read data: %w", err)
	}

	err = json.Unmarshal(data, a)
	if err != nil {
		return fmt.Errorf("failed to unmarshal data: %w", err)
	}

	return nil
}
