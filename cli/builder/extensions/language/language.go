package language

import (
	"context"

	"github.com/agntcy/dir/cli/types"
)

const (
	// ExtensionName is the corresponding OASF feature name
	ExtensionName = "oasf.agntcy.org/features/runtime/language"
	// ExtensionVersion is the version of extension
	ExtensionVersion = "v0.0.0"
)

type Config struct {
	Type    string `yaml:"type"`
	Version string `yaml:"version"`
}

type language struct {
	Type    string
	Version string
}

func New(cfg *Config) *language {
	return &language{
		Type:    cfg.Type,
		Version: cfg.Version,
	}
}

func (l *language) Build(_ context.Context) (*types.AgentExtension, error) {
	return &types.AgentExtension{
		Name:    ExtensionName,
		Version: ExtensionVersion,
		Specs: map[string]string{
			"type":    l.Type,
			"version": l.Version,
		},
	}, nil
}
