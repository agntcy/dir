package processor

import (
	"fmt"

	"github.com/agntcy/dir/discovery/pkg/processor/config"
	"github.com/agntcy/dir/discovery/pkg/processor/health"
	"github.com/agntcy/dir/discovery/pkg/processor/openapi"
	"github.com/agntcy/dir/discovery/pkg/types"
)

func NewProcessors(cfg config.Config) ([]types.WorkloadProcessor, error) {
	var processors []types.WorkloadProcessor

	// Create processors based on configuration
	if cfg.Health.Enabled {
		processors = append(processors, health.NewProcessor(cfg.Health))
	}

	if cfg.OpenAPI.Enabled {
		processors = append(processors, openapi.NewProcessor(cfg.OpenAPI))
	}

	// Validate created processors
	if len(processors) == 0 {
		return nil, fmt.Errorf("no processors enabled")
	}

	return processors, nil
}
