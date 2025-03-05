package config

import (
	"fmt"
	"io"
	"os"

	"gopkg.in/yaml.v2"

	apicore "github.com/agntcy/dir/api/core/v1alpha1"
	"github.com/agntcy/dir/cli/builder/extensions/framework"
	"github.com/agntcy/dir/cli/builder/extensions/language"
	"github.com/agntcy/dir/cli/builder/extensions/skills"
)

type Locator struct {
	Type string `yaml:"type"`
	URL  string `yaml:"url"`
}

type Extension struct {
	Name    string         `yaml:"name"`
	Version string         `yaml:"version"`
	Specs   map[string]any `yaml:"specs"`
}

type Config struct {
	Name       string      `yaml:"name"`
	Version    string      `yaml:"version"`
	Authors    []string    `yaml:"authors"`
	Locators   []Locator   `yaml:"locators"`
	Extensions []Extension `yaml:"extensions"`

	Source       string   `yaml:"source"`
	SourceIgnore []string `yaml:"source-ignore"`

	LLMAnalyzer bool `yaml:"llmanalyzer"`
	CrewAI      bool `yaml:"crewai"`

	Framework framework.Config `yaml:"framework"`
	Language  language.Config  `yaml:"language"`
	Skills    skills.Config    `yaml:"skills"`
}

func (c *Config) LoadFromFile(path string) error {
	reader, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}

	data, err := io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("failed to read data: %w", err)
	}

	err = yaml.Unmarshal(data, c)
	if err != nil {
		return fmt.Errorf("failed to unmarshal data: %w", err)
	}

	return nil
}

func (c *Config) GetAPILocators() ([]*apicore.Locator, error) {
	var locators []*apicore.Locator
	for _, locator := range c.Locators {
		var ok bool
		var locatorType int32
		if locatorType, ok = apicore.LocatorType_value[locator.Type]; !ok {
			return nil, fmt.Errorf("invalid locator type: %s", locator.Type)
		}

		locators = append(locators, &apicore.Locator{
			Type: apicore.LocatorType(locatorType),
			Source: &apicore.LocatorSource{
				Url: locator.URL,
			},
		})
	}

	return locators, nil
}

func (c *Config) Validate() error {
	switch framework.FrameworkType(c.Framework.Type) {
	case framework.CrewAI, framework.Autogen, framework.Llmaindex, framework.Langchain:
	default:
		return fmt.Errorf("invalid framework type: %s", c.Framework.Type)
	}

	return nil
}
