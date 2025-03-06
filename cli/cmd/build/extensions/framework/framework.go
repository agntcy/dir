package framework

import (
	"context"
	"fmt"

	"github.com/agntcy/dir/cli/types"
	"gopkg.in/yaml.v2"
)

const (
	// ExtensionName is the corresponding OASF feature name
	ExtensionName = "oasf.agntcy.org/features/runtime/framework"
	// ExtensionVersion is the version of extension
	ExtensionVersion = "v0.0.0"
)

type FrameworkType string

const (
	CrewAI    FrameworkType = "crewai"
	Autogen   FrameworkType = "autogen"
	Llmaindex FrameworkType = "llma-index"
	Langchain FrameworkType = "langchain"
)

type Config struct {
	Type    FrameworkType `yaml:"type"`
	Version string        `yaml:"version"`
}

func (c *Config) From(data map[string]any) error {
	yamlData, err := yaml.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	err = yaml.Unmarshal(yamlData, c)
	if err != nil {
		return fmt.Errorf("failed to unmarshal data: %w", err)
	}

	return nil
}

type framework struct {
	Type    FrameworkType
	Version string
}

func New(cfg *Config) *framework {
	return &framework{
		Type:    cfg.Type,
		Version: cfg.Version,
	}
}

func (fw *framework) Build(_ context.Context) (*types.AgentExtension, error) {
	return &types.AgentExtension{
		Name:    ExtensionName,
		Version: ExtensionVersion,
		Specs: map[string]string{
			"type":    string(fw.Type),
			"version": fw.Version,
		},
	}, nil
}
