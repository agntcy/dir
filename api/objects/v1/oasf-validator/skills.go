// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:revive,wsl,stylecheck,nlreturn,mnd,gofumpt
package oasfvalidator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	objectsv1 "github.com/agntcy/dir/api/objects/v1"
)

const OASFEndpoint = "https://schema.oasf.agntcy.org/api/skills"

var Validator = &SkillValidator{
	skills: make(map[uint64]*objectsv1.Skill),
}

type SkillValidator struct {
	skills map[uint64]*objectsv1.Skill
	loaded bool
	mu     sync.RWMutex
}

func init() {
	if err := Validator.fetchSkills(); err != nil {
		log.Printf("failed to fetch skills from OASF: %v", err)
	}

	Validator.loaded = true
}

func (sv *SkillValidator) fetchSkills() error {
	// Create a context with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create a new HTTP request with the context
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, OASFEndpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Perform the HTTP request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch skills from OASF: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	var rawSkills []map[string]any
	if err := json.Unmarshal(body, &rawSkills); err != nil {
		return fmt.Errorf("failed to unmarshal skills: %w", err)
	}

	for _, rawSkill := range rawSkills {
		uid, ok := rawSkill["uid"].(float64)
		if !ok {
			return errors.New("invalid uid type for skill")
		}

		// Skip the base class
		if uid == 0 {
			continue
		}

		categoryUid, ok := rawSkill["category_uid"].(float64)
		if !ok {
			return fmt.Errorf("invalid category_uid type for skill %v", uid)
		}

		categoryName, ok := rawSkill["category_name"].(string)
		if !ok {
			return fmt.Errorf("invalid category_name type for skill %v", uid)
		}

		className, ok := rawSkill["caption"].(string)
		if !ok {
			return fmt.Errorf("invalid caption type for skill %v", uid)
		}

		skill := objectsv1.Skill{
			ClassUid:     uint64(uid),
			CategoryUid:  uint64(categoryUid),
			CategoryName: &categoryName,
			ClassName:    &className,
		}

		sv.mu.Lock()
		sv.skills[skill.GetClassUid()] = &skill
		sv.mu.Unlock()
	}

	return nil
}

func (sv *SkillValidator) HasSkill(classUid uint64) bool {
	sv.mu.RLock()
	defer sv.mu.RUnlock()
	_, exists := sv.skills[classUid]

	return exists
}

func (sv *SkillValidator) GetSkill(classUid uint64) *objectsv1.Skill {
	sv.mu.RLock()
	defer sv.mu.RUnlock()

	return sv.skills[classUid]
}

func (sv *SkillValidator) GetSkillByName(className string) *objectsv1.Skill {
	sv.mu.RLock()
	defer sv.mu.RUnlock()
	for _, skill := range sv.skills {
		if skill.ClassName != nil && *skill.ClassName == className { //nolint:protogetter
			return skill
		}
	}

	return nil
}

func (sv *SkillValidator) Validate(skills []*objectsv1.Skill) error {
	var errorMessages []string

	for _, skill := range skills {
		if skill.GetClassUid() == 0 {
			errorMessages = append(errorMessages, fmt.Sprintf("'class_uid' is required for skill %+v.", skill))
			continue
		}

		if !sv.HasSkill(skill.GetClassUid()) {
			errorMessages = append(errorMessages, fmt.Sprintf("'class_uid' not found for skill %+v.", skill))
		}

		//TODO: Check if uid actually belongs to the provided class name, category uid and category name
	}

	if len(errorMessages) > 0 {
		return fmt.Errorf("validation failed with the following errors:\n%v", strings.Join(errorMessages, "\n"))
	}

	return nil
}

func ValidateSkills(skills []*objectsv1.Skill) error {
	return Validator.Validate(skills)
}
