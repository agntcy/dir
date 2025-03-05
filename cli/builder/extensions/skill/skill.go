package skill

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

type skill struct {
	skills []string
}

func New(skills []string) *skill {
	return &skill{
		skills: skills,
	}
}

func (a *skill) Build(_ context.Context) (*types.AgentExtension, error) {
	return &types.AgentExtension{
		Name:    ExtensionName,
		Version: ExtensionVersion,
		Specs: map[string][]string{
			"skills": a.skills,
		},
	}, nil
}
