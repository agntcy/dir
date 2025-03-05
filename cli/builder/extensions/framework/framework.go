package framework

import (
	"context"

	"github.com/agntcy/dir/cli/builder/config"
	"github.com/agntcy/dir/cli/types"
)

const (
	// ExtensionName is the corresponding OASF feature name
	ExtensionName = "oasf.agntcy.org/features/runtime/framework"
	// ExtensionVersion is the version of extension
	ExtensionVersion = "v0.0.0"
)

type framework struct {
	Type    string
	Version string
}

func New(cfg *config.Framework) *framework {
	return &framework{
		Type:    cfg.Type,
		Version: cfg.Version,
	}
}

func (a *framework) Build(_ context.Context) (*types.AgentExtension, error) {
	return &types.AgentExtension{
		Name:    ExtensionName,
		Version: ExtensionVersion,
		Specs: map[string]string{
			"type":    a.Type,
			"version": a.Version,
		},
	}, nil
}
