package config

import (
	"fmt"
	"io"
	"os"
	"strings"

	"gopkg.in/yaml.v2"

	apicore "github.com/agntcy/dir/api/core/v1alpha1"
	builderconfig "github.com/agntcy/dir/cli/builder/config"
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

	Builder builderconfig.Config `yaml:"builder"`
}

func (c *Config) LoadFromFlags(name, version string, llmAnalyzer, crewai bool, authors, rawLocators []string) error {
	c.Name = name
	c.Version = version
	c.Authors = authors

	c.Builder.LLMAnalyzer = llmAnalyzer
	c.Builder.CrewAI = crewai

	// Load in locators
	var locators []Locator
	for _, locator := range rawLocators {
		// Split locator into type and URL
		parts := strings.SplitN(locator, ":", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid locator format, expected 'type:url'")
		}

		locators = append(locators, Locator{
			Type: parts[0],
			URL:  parts[1],
		})
	}
	c.Locators = locators

	// TODO Allow for extensions to be passed in via flags?

	return nil
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

func (c *Config) Merge(extra *Config) {
	c.Name = firstNonEmpty(c.Name, extra.Name)
	c.Version = firstNonEmpty(c.Version, extra.Version)
	// c.Builder.LLMAnalyzer = c.Builder.LLMAnalyzer
	// c.Builder.CrewAI = c.Builder.CrewAI
	// TODO check if slice fields should be merged or replaced
	c.Authors = firstNonEmptySlice(c.Authors, extra.Authors)
	c.Locators = firstNonEmptySlice(c.Locators, extra.Locators)
	c.Extensions = firstNonEmptySlice(c.Extensions, extra.Extensions)

	c.Builder.Source = firstNonEmpty(c.Builder.Source, extra.Builder.Source)
	c.Builder.SourceIgnore = firstNonEmptySlice(c.Builder.SourceIgnore, extra.Builder.SourceIgnore)
}

func firstNonEmpty(opt, cfg string) string {
	if opt != "" {
		return opt
	}
	return cfg
}

func firstNonEmptySlice[T any](opt, cfg []T) []T {
	if len(opt) > 0 {
		return opt
	}
	return cfg
}
