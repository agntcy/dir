// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package objectsv1

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

//nolint:gocognit,cyclop
func (x *Agent) Merge(other *Agent) {
	if other == nil {
		return
	}

	// Only use other's scalar fields if receiver doesn't have them set
	x.Name = firstNonEmptyString(x.GetName(), other.GetName())
	x.Version = firstNonEmptyString(x.GetVersion(), other.GetVersion())
	x.Description = firstNonEmptyString(x.GetDescription(), other.GetDescription())

	if x.GetCreatedAt() == "" {
		x.CreatedAt = other.GetCreatedAt()
	}

	// Merge slices without duplicates, keeping receiver's values first
	if len(other.GetAuthors()) > 0 {
		x.Authors = removeDuplicates(append(other.GetAuthors(), x.GetAuthors()...))
	}

	// Merge annotations, keeping receiver's values when keys conflict
	if x.GetAnnotations() == nil {
		x.Annotations = make(map[string]string)
	}

	for k, v := range other.GetAnnotations() {
		if _, exists := x.GetAnnotations()[k]; !exists {
			x.Annotations[k] = v
		}
	}

	// Merge Locators, keeping receiver's values when "type/url" conflict
	if len(other.GetLocators()) > 0 {
		x.Locators = mergeItems(
			x.GetLocators(),
			other.GetLocators(),
			func(locator *Locator) string {
				return locator.Key()
			},
		)
	}

	// Merge Extensions, keeping receiver's values when "name/version" conflict
	if len(other.GetExtensions()) > 0 {
		x.Extensions = mergeItems(
			x.GetExtensions(),
			other.GetExtensions(),
			func(extension *Extension) string {
				return extension.Key()
			},
		)
	}

	// Merge skills, keeping receiver's values when "key" conflict
	if len(other.GetSkills()) > 0 {
		x.Skills = mergeItems(
			x.GetSkills(),
			other.GetSkills(),
			func(skill *Skill) string {
				return skill.Key()
			},
		)
	}
}

func (x *Agent) LoadFromReader(reader io.Reader) ([]byte, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read data: %w", err)
	}

	err = json.Unmarshal(data, x)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal data: %w", err)
	}

	return data, nil
}

func (x *Agent) LoadFromFile(path string) ([]byte, error) {
	reader, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	return x.LoadFromReader(reader)
}

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
