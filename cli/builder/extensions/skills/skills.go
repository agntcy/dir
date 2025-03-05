package skills

import (
	"context"

	"github.com/agntcy/dir/cli/types"
)

const (
	// ExtensionName is the corresponding OASF feature name
	ExtensionName = "oasf.agntcy.org/skills"
	// ExtensionVersion is the version of extension
	ExtensionVersion = "v0.0.0"
)

type Config struct {
	Items []string `yaml:"items"`
}

type skills struct {
	skills []string
}

func New(cfg *Config) *skills {
	return &skills{
		skills: cfg.Items,
	}
}

func (s *skills) Build(_ context.Context) (*types.AgentExtension, error) {
	return &types.AgentExtension{
		Name:    ExtensionName,
		Version: ExtensionVersion,
		Specs: map[string][]string{
			"skills": s.skills,
		},
	}, nil
}
