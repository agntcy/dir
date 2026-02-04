// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package runtime

import (
	"fmt"

	"github.com/agntcy/dir/runtime/discovery/runtime/config"
	"github.com/agntcy/dir/runtime/discovery/runtime/docker"
	"github.com/agntcy/dir/runtime/discovery/runtime/k8s"
	"github.com/agntcy/dir/runtime/discovery/types"
)

//nolint:wrapcheck
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
