package config

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v2"

	apicore "github.com/agntcy/dir/api/core/v1alpha1"
)

type Artifact struct {
	Type string `yaml:"type"`
	URL  string `yaml:"url"`
}

type Extension struct {
	Name    string         `yaml:"name"`
	Version string         `yaml:"version"`
	Specs   map[string]any `yaml:"specs"`
}

type Config struct {
	Source      string      `yaml:"source"`
	Name        string      `yaml:"name"`
	Version     string      `yaml:"version"`
	CreatedAt   time.Time   `yaml:"created_at"`
	LLMAnalyzer bool        `yaml:"llmanalyzer"`
	Authors     []string    `yaml:"authors"`
	Categories  []string    `yaml:"categories"`
	Artifacts   []Artifact  `yaml:"artifacts"`
	Extensions  []Extension `yaml:"extensions"`
}

func (c *Config) LoadFromFlags(name, version, createdAt string, llmAnalyzer bool, authors, categories []string, rawArtifacts []string) error {
	c.Name = name
	c.Version = version
	c.LLMAnalyzer = llmAnalyzer
	c.Authors = authors
	c.Categories = categories

	// Override creation time if requested
	c.CreatedAt = time.Now()
	if createdAt != "" {
		var err error
		c.CreatedAt, err = time.Parse(time.RFC3339, createdAt)
		if err != nil {
			return fmt.Errorf("failed to parse create time: %w", err)
		}
	}

	// Load in artifacts
	var artifacts []Artifact
	for _, artifact := range rawArtifacts {
		// Split artifact into type and URL
		parts := strings.SplitN(artifact, ":", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid artifact format, expected 'type:url'")
		}

		artifacts = append(artifacts, Artifact{
			Type: parts[0],
			URL:  parts[1],
		})
	}
	c.Artifacts = artifacts

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
	for _, locator := range c.Artifacts {
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
	// c.LLMAnalyzer = c.LLMAnalyzer
	// TODO check if slice fields should be merged or replaced
	c.Authors = firstNonEmptySlice(c.Authors, extra.Authors)
	c.Categories = firstNonEmptySlice(c.Categories, extra.Categories)
	c.Artifacts = firstNonEmptySlice(c.Artifacts, extra.Artifacts)
	c.Extensions = firstNonEmptySlice(c.Extensions, extra.Extensions)
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
