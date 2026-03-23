// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package test_utils

import (
	"encoding/json"
	"fmt"
	"slices"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/onsi/gomega"
	"google.golang.org/protobuf/types/known/structpb"
)

var modules = []Module{
	{
		Name: "integration/mcp",
		Data: MCPData{
			Name: "github-mcp-server",
			Connections: []MCPServerConnection{
				{
					Type:    "stdio",
					Command: "docker",
					Args: []string{
						"run",
						"-i",
						"--rm",
						"-e",
						"GITHUB_PERSONAL_ACCESS_TOKEN",
						"ghcr.io/github/github-mcp-server",
					},
					EnvVars: []EnvironmentVariable{
						{
							Name:         "GITHUB_PERSONAL_ACCESS_TOKEN",
							DefaultValue: "",
							Description:  "Secret value for GITHUB_PERSONAL_ACCESS_TOKEN",
						},
					},
				},
			},
		},
	},
	{
		Name: "integration/a2a",
		Data: A2AData{
			CardSchemaVersion: "v1.0.0",
			// TODO: Add A2A Card faker
			CardData: map[string]any{
				"protocolVersions": []string{"0.2.6"},
				"name":             "burger_seller_agent",
				"description":      "Helps with creating burger orders",
			},
		},
	},
}

var skills = []Skill{
	{Id: 101, Name: "natural_language_processing/natural_language_understanding"},                            //nolint:mnd
	{Id: 10101, Name: "natural_language_processing/natural_language_understanding/contextual_comprehension"}, //nolint:mnd
	{Id: 10102, Name: "natural_language_processing/natural_language_understanding/semantic_understanding"},   //nolint:mnd
	{Id: 108, Name: "natural_language_processing/ethical_interaction"},                                       //nolint:mnd
	{Id: 201, Name: "images_computer_vision/image_segmentation"},                                             //nolint:mnd
	{Id: 1504, Name: "advanced_reasoning_planning/hypothesis_generation"},                                    //nolint:mnd
}

var GofakeitOASF100Lookups = map[string]gofakeit.Info{
	"oasf100skills": {
		Display:     "OASF100Skills",
		Category:    "oasf",
		Description: "OASF 1.0.0 skills (a slice of 1 to 5 skills)",
		Output:      "Skills",
		Generate: func(f *gofakeit.Faker, m *gofakeit.MapParams, info *gofakeit.Info) (any, error) {
			skillsCopy := make(Skills, len(skills))
			copy(skillsCopy, skills)

			n := f.Number(1, 5) //nolint:mnd
			f.ShuffleAnySlice(skillsCopy)

			return skillsCopy[:n], nil
		},
	},
	"oasf100modules": {
		Display:     "OASF100Modules",
		Category:    "oasf",
		Description: "OASF 1.0.0 modules (a slice of 1 to 2 modules)",
		Output:      "Modules",
		Generate: func(f *gofakeit.Faker, m *gofakeit.MapParams, info *gofakeit.Info) (any, error) {
			modulesCopy := make(Modules, len(modules))
			copy(modulesCopy, modules)

			n := f.Number(1, 2) //nolint:mnd
			f.ShuffleAnySlice(modulesCopy)

			return modulesCopy[:n], nil
		},
	},
	"oasf100recordname": {
		Display:     "OASF100RecordName",
		Category:    "oasf",
		Description: "OASF 1.0.0 record name",
		Example:     "Marketing Strategy Agent",
		Output:      "string",
		Generate: func(f *gofakeit.Faker, m *gofakeit.MapParams, info *gofakeit.Info) (any, error) {
			return fmt.Sprintf("%s Agent", gofakeit.JobTitle()), nil
		},
	},
	"oasf100authors": {
		Display:     "OASF100Authors",
		Category:    "oasf",
		Description: "OASF 1.0.0 authors",
		Output:      "[]string",
		Generate: func(f *gofakeit.Faker, m *gofakeit.MapParams, info *gofakeit.Info) (any, error) {
			n := f.Number(1, 5) //nolint:mnd
			authors := []string{}

			for range n {
				author := f.Company()
				if !slices.Contains(authors, author) {
					authors = append(authors, author)
				}
			}

			return authors, nil
		},
	},
}

// Skill represents an OASF 1.0.0 Skill category
// https://schema.oasf.outshift.com/1.0.0/skill_categories
type Skill struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
}

// The Skills type alias is necessary for Gofakeit to work
// (Gofakeit doesn't work with anonymous slices).
type Skills []Skill

// Locator represents an OASF 1.0.0 Locator object
// https://schema.oasf.outshift.com/1.0.0/objects/locator
type Locator struct {
	LocatorType string   `fake:"{randomstring:[binary,container_image,helm_chart,package,source_code,unspecified,url]}" json:"type"`
	Urls        []string `fake:"{url}"                                                                                  json:"urls"`
}

// Module represents an OASF 1.0.0 Module category
// https://schema.oasf.outshift.com/1.0.0/module_categories
type Module struct {
	Name string `json:"name"`
	Data any    `json:"data"`
}

// The Modules type alias is necessary for Gofakeit to work
// (Gofakeit doesn't work with anonymous slices).
type Modules []Module

// MCPData represents an OASF 1.0.0 MCP Data object
// https://schema.oasf.outshift.com/1.0.0/objects/mcp_data
type MCPData struct {
	Name        string                `json:"name"`
	Connections []MCPServerConnection `json:"connections"`
}

// MCPServerConnection represents an OASF 1.0.0 MCP Server Connection object
// https://schema.oasf.outshift.com/1.0.0/objects/mcp_server_connection
type MCPServerConnection struct {
	Type    string                `fake:"{randomstring:[sse,stdio,streamable-http]}" json:"type"`
	Args    []string              `json:"args"`
	EnvVars []EnvironmentVariable `json:"env_vars"`
	Headers map[string]string     `json:"headers"`
	Command string                `json:"command,omitempty"`
	Url     string                `json:"url,omitempty"`
}

// EnvironmentVariable represents an OASF 1.0.0 Environment Variable object
// https://schema.oasf.outshift.com/1.0.0/objects/env_var
type EnvironmentVariable struct {
	Name         string `json:"name"`
	Description  string `json:"description"`
	DefaultValue string `json:"default_value"`
	Required     bool   `json:"required"`
}

// A2AData represents an OASF 1.0.0 A2A Data object
// https://schema.oasf.outshift.com/1.0.0/objects/a2a_data
type A2AData struct {
	CardData          any    `json:"card_data"`
	CardSchemaVersion string `fake:"{appversion}" json:"card_schema_version"`
}

// Record represents an OASF 1.0.0 Record object
// https://schema.oasf.outshift.com/1.0.0/objects/record
type Record struct {
	Name          string    `fake:"{oasf100recordname}"  json:"name"`
	SchemaVersion string    `fake:"1.0.0"                json:"schema_version"`
	Version       string    `fake:"{appversion}"         json:"version"`
	Description   string    `fake:"{productdescription}" json:"description"`
	Authors       []string  `fake:"{oasf100authors}"     json:"authors"`
	CreatedAt     string    `fake:"{pastdaterfc3339}"    json:"created_at"`
	Skills        Skills    `fake:"{oasf100skills}"      json:"skills"`
	Locators      []Locator `fakesize:"1"                json:"locators"`
	Modules       Modules   `fake:"{oasf100modules}"     json:"modules"`
}

func FakeRecord() *Record {
	var record Record

	err := gofakeit.Struct(&record)
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	return &record
}

func (r *Record) PbStruct() *structpb.Struct {
	bytes, err := json.Marshal(r)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	var record map[string]*structpb.Value

	err = json.Unmarshal(bytes, &record)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	return &structpb.Struct{Fields: record}
}
