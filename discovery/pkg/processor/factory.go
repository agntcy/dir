package processor

import (
	"fmt"

	"github.com/agntcy/dir/discovery/pkg/processor/a2a"
	"github.com/agntcy/dir/discovery/pkg/processor/config"
	"github.com/agntcy/dir/discovery/pkg/processor/oasf"
	"github.com/agntcy/dir/discovery/pkg/types"
)

func NewProcessors(cfg config.Config) ([]types.WorkloadProcessor, error) {
	var processors []types.WorkloadProcessor

	// Create processors based on configuration
	if cfg.A2A.Enabled {
		processors = append(processors, a2a.NewProcessor(cfg.A2A))
	}

	// Create OASF processor
	if cfg.OASF.Enabled {
		oasfProc, err := oasf.NewProcessor(cfg.OASF)
		if err != nil {
			return nil, fmt.Errorf("failed to create OASF processor: %w", err)
		}
		processors = append(processors, oasfProc)
	}

	// Validate created processors
	if len(processors) == 0 {
		return nil, fmt.Errorf("no processors enabled")
	}

	return processors, nil
}
