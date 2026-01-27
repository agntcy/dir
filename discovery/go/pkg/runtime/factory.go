package runtime

import (
	"fmt"

	"github.com/agntcy/dir/discovery/pkg/runtime/config"
	"github.com/agntcy/dir/discovery/pkg/runtime/docker"
	"github.com/agntcy/dir/discovery/pkg/runtime/k8s"
	"github.com/agntcy/dir/discovery/pkg/types"
)

func NewAdapter(cfg config.Config) (types.RuntimeAdapter, error) {
	switch cfg.Type {
	case docker.RuntimeType:
		return docker.NewAdapter(cfg.Docker)
	case k8s.RuntimeType:
		return k8s.NewAdapter(cfg.Kubernetes)
	default:
		return nil, fmt.Errorf("unsupported runtime: %s", cfg.Type)
	}
}
