// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package types

import (
	"encoding/json"
	"fmt"
	coretypes "github.com/agntcy/dir/api/core/v1alpha1"
	"github.com/agntcy/dir/utils"
	"gorm.io/gorm"
)

type QueryFilters struct {
	Limit  int `json:"limit,omitempty"`
	Offset int `json:"offset,omitempty"`

	Name        string   `json:"name,omitempty"`
	Version     string   `json:"version,omitempty"`
	Description string   `json:"description,omitempty"`
	Authors     []string `json:"authors,omitempty"`

	SkillNames      []string `json:"skill_names,omitempty"`
	SkillCategories []string `json:"skill_categories,omitempty"`

	LocatorTypes []string `json:"locator_types,omitempty"`

	ExtensionNames    []string `json:"extension_names,omitempty"`
	ExtensionVersions []string `json:"extension_versions,omitempty"`
}

type SearchAPI interface {
	// AddAgent adds an object to the search database.
	AddAgent(agent *coretypes.Agent) error

	// QueryAgents queries agents from the search database.
	QueryAgents(filters QueryFilters) ([]*coretypes.Agent, error)
}

type Agent struct {
	gorm.Model
	Name          string `gorm:"not null"`
	Version       string `gorm:"not null"`
	SchemaVersion string `gorm:"not null"`
	Description   string `gorm:"not null"`
	Authors       string `gorm:"not null"`
	Annotations   string `gorm:"not null"`
}

func (a *Agent) FromCoreAgent(coreAgent *coretypes.Agent) error {
	a.Name = coreAgent.Name
	a.Version = coreAgent.Version
	a.SchemaVersion = coreAgent.SchemaVersion
	a.Description = coreAgent.Description

	authorsJSON, err := json.Marshal(coreAgent.Authors)
	if err != nil {
		return fmt.Errorf("failed to serialize Authors: %w", err)
	}
	a.Authors = string(authorsJSON)

	annotationsJSON, err := json.Marshal(coreAgent.Annotations)
	if err != nil {
		return fmt.Errorf("failed to serialize Annotations: %w", err)
	}
	a.Annotations = string(annotationsJSON)

	return nil
}

func (a *Agent) ToCoreAgent() (*coretypes.Agent, error) {
	coreAgent := &coretypes.Agent{
		Name:          a.Name,
		Version:       a.Version,
		SchemaVersion: a.SchemaVersion,
		Description:   a.Description,
	}

	if a.Authors != "" {
		if err := json.Unmarshal([]byte(a.Authors), &coreAgent.Authors); err != nil {
			return nil, fmt.Errorf("failed to deserialize Authors: %w", err)
		}
	}

	if a.Annotations != "" {
		if err := json.Unmarshal([]byte(a.Annotations), &coreAgent.Annotations); err != nil {
			return nil, fmt.Errorf("failed to deserialize Annotations: %w", err)
		}
	}

	return coreAgent, nil
}

type Skill struct {
	gorm.Model
	AgentID      uint   `gorm:"not null;index"`
	CategoryUID  uint64 `gorm:"not null"`
	CategoryName string `gorm:"not null"`
	ClassUID     uint64 `gorm:"not null"`
	ClassName    string `gorm:"not null"`
	Annotations  string `gorm:"not null"`
}

func (s *Skill) FromCoreSkill(coreSkill *coretypes.Skill, agentID uint) error {
	s.AgentID = agentID
	s.CategoryUID = coreSkill.CategoryUid
	s.ClassUID = coreSkill.ClassUid

	if coreSkill.CategoryName != nil {
		s.CategoryName = *coreSkill.CategoryName
	}

	if coreSkill.ClassName != nil {
		s.ClassName = *coreSkill.ClassName
	}

	annotationsJSON, err := json.Marshal(coreSkill.Annotations)
	if err != nil {
		return fmt.Errorf("failed to serialize Annotations: %w", err)
	}
	s.Annotations = string(annotationsJSON)

	return nil
}

func (s *Skill) ToCoreSkill() (*coretypes.Skill, error) {
	coreSkill := &coretypes.Skill{
		CategoryUid:  s.CategoryUID,
		ClassUid:     s.ClassUID,
		CategoryName: utils.ToPtr(s.CategoryName),
		ClassName:    utils.ToPtr(s.ClassName),
	}

	if s.Annotations != "" {
		if err := json.Unmarshal([]byte(s.Annotations), &coreSkill.Annotations); err != nil {
			return nil, fmt.Errorf("failed to deserialize Annotations: %w", err)
		}
	}

	return coreSkill, nil
}

type Locator struct {
	gorm.Model
	AgentID     uint   `gorm:"not null;index"`
	Type        string `gorm:"not null"`
	URL         string `gorm:"not null"`
	Size        uint64 `gorm:"not null"`
	Digest      string `gorm:"not null"`
	Annotations string `gorm:"not null"`
}

func (l *Locator) FromCoreLocator(locator *coretypes.Locator, agentID uint) error {
	l.AgentID = agentID
	l.Type = locator.Type
	l.URL = locator.Url

	if locator.Size != nil {
		l.Size = *locator.Size
	}

	if locator.Digest != nil {
		l.Digest = *locator.Digest
	}

	annotationsJSON, err := json.Marshal(locator.Annotations)
	if err != nil {
		return fmt.Errorf("failed to serialize Annotations: %w", err)
	}
	l.Annotations = string(annotationsJSON)

	return nil
}

func (l *Locator) ToCoreLocator() (*coretypes.Locator, error) {
	coreLocator := &coretypes.Locator{
		Type:   l.Type,
		Url:    l.URL,
		Size:   utils.ToPtr(l.Size),
		Digest: utils.ToPtr(l.Digest),
	}

	if l.Annotations != "" {
		if err := json.Unmarshal([]byte(l.Annotations), &coreLocator.Annotations); err != nil {
			return nil, fmt.Errorf("failed to deserialize Annotations: %w", err)
		}
	}

	return coreLocator, nil
}

type Signature struct {
	gorm.Model
	AgentID       uint   `gorm:"not null;index"`
	SignedAt      string `gorm:"not null"`
	Algorithm     string `gorm:"not null"`
	Signature     string `gorm:"not null"`
	Certificate   string `gorm:"not null"`
	ContentType   string `gorm:"not null"`
	ContentBundle string `gorm:"not null"`
}

func (s *Signature) FromCoreSignature(signature *coretypes.Signature, agentID uint) error {
	s.AgentID = agentID
	s.SignedAt = signature.SignedAt
	s.Algorithm = signature.Algorithm
	s.Signature = signature.Signature
	s.Certificate = signature.Certificate
	s.ContentType = signature.ContentType
	s.ContentBundle = signature.ContentBundle

	return nil
}

func (s *Signature) ToCoreSignature() *coretypes.Signature {
	return &coretypes.Signature{
		SignedAt:      s.SignedAt,
		Algorithm:     s.Algorithm,
		Signature:     s.Signature,
		Certificate:   s.Certificate,
		ContentType:   s.ContentType,
		ContentBundle: s.ContentBundle,
	}
}

type Extension struct {
	gorm.Model
	AgentID     uint   `gorm:"not null;index"`
	Name        string `gorm:"not null"`
	Version     string `gorm:"not null"`
	Data        string `gorm:"not null"`
	Annotations string `gorm:"not null"`
}

func (e *Extension) FromCoreExtension(ext *coretypes.Extension, agentID uint) error {
	e.AgentID = agentID
	e.Name = ext.Name
	e.Version = ext.Version

	dataJSON, err := json.Marshal(ext.Data)
	if err != nil {
		return fmt.Errorf("failed to serialize Data: %w", err)
	}
	e.Data = string(dataJSON)

	annotationsJSON, err := json.Marshal(ext.Annotations)
	if err != nil {
		return fmt.Errorf("failed to serialize Annotations: %w", err)
	}
	e.Annotations = string(annotationsJSON)

	return nil
}

func (e *Extension) ToCoreExtension() (*coretypes.Extension, error) {
	coreExtension := &coretypes.Extension{
		Name:    e.Name,
		Version: e.Version,
	}

	if e.Data != "" {
		var data map[string]interface{}
		if err := json.Unmarshal([]byte(e.Data), &data); err != nil {
			return nil, fmt.Errorf("failed to deserialize Data: %w", err)
		}

		structData, err := utils.ToStructpb(data)
		if err != nil {
			return nil, fmt.Errorf("failed to convert data to structpb: %w", err)
		}
		coreExtension.Data = structData
	}

	if e.Annotations != "" {
		if err := json.Unmarshal([]byte(e.Annotations), &coreExtension.Annotations); err != nil {
			return nil, fmt.Errorf("failed to deserialize Annotations: %w", err)
		}
	}

	return coreExtension, nil
}
