// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package corev1alpha1

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	objectsv1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/objects/v1"
)

type Agent struct {
	*objectsv1.Agent
}

func (a *Agent) GetLocators() []*Locator {
	if a.Agent == nil {
		return nil
	}

	locators := make([]*Locator, len(a.Agent.GetLocators()))
	for i, locator := range a.Agent.GetLocators() {
		locators[i] = &Locator{Locator: locator}
	}

	return locators
}

func (a *Agent) GetExtensions() []*Extension {
	if a.Agent == nil {
		return nil
	}

	extensions := make([]*Extension, len(a.Agent.GetExtensions()))
	for i, extension := range a.Agent.GetExtensions() {
		extensions[i] = &Extension{Extension: extension}
	}

	return extensions
}

func (a *Agent) GetSkills() []*Skill {
	if a.Agent == nil {
		return nil
	}

	skills := make([]*Skill, len(a.Agent.GetSkills()))
	for i, skill := range a.Agent.GetSkills() {
		skills[i] = &Skill{Skill: skill}
	}

	return skills
}

//nolint:gocognit,cyclop
func (a *Agent) Merge(other *Agent) {
	if other == nil {
		return
	}

	// Only use other's scalar fields if receiver doesn't have them set
	a.Name = firstNonEmptyString(a.GetName(), other.GetName())
	a.Version = firstNonEmptyString(a.GetVersion(), other.GetVersion())
	a.Description = firstNonEmptyString(a.GetDescription(), other.GetDescription())

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
		wrappedLocators := mergeItems(
			a.GetLocators(),
			other.GetLocators(),
			func(locator *Locator) string {
				return locator.Key()
			},
		)

		locators := make([]*objectsv1.Locator, len(wrappedLocators))

		for i, locator := range wrappedLocators {
			if locator == nil {
				continue
			}

			locators[i] = locator.ToOASFSchema()
		}

		a.Locators = locators
	}

	// Merge Extensions, keeping receiver's values when "name/version" conflict
	if len(other.GetExtensions()) > 0 {
		wrappedExtensions := mergeItems(
			a.GetExtensions(),
			other.GetExtensions(),
			func(extension *Extension) string {
				return extension.Key()
			},
		)

		extensions := make([]*objectsv1.Extension, len(wrappedExtensions))

		for i, extension := range wrappedExtensions {
			if extension == nil {
				continue
			}

			extensions[i] = extension.ToOASFSchema()
		}

		a.Extensions = extensions
	}

	// Merge skills, keeping receiver's values when "key" conflict
	if len(other.GetSkills()) > 0 {
		wrappedSkills := mergeItems(
			a.GetSkills(),
			other.GetSkills(),
			func(skill *Skill) string {
				return skill.Key()
			},
		)

		skills := make([]*objectsv1.Skill, len(wrappedSkills))

		for i, skill := range wrappedSkills {
			if skill == nil {
				continue
			}

			skills[i] = skill.ToOASFSchema()
		}

		a.Skills = skills
	}
}

func (a *Agent) LoadFromReader(reader io.Reader) ([]byte, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read data: %w", err)
	}

	err = json.Unmarshal(data, a)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal data: %w", err)
	}

	return data, nil
}

func (a *Agent) LoadFromFile(path string) ([]byte, error) {
	reader, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	return a.LoadFromReader(reader)
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
