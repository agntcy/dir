package types

import (
	"context"

	runtimev1 "github.com/agntcy/dir/api/runtime/v1"
)

type DiscoveryAPI interface {
	ListProcesses(
		ctx context.Context,
		req *runtimev1.ListProcessesRequest,
		fn func(*runtimev1.Process) error,
	) error

	IsReady(context.Context) bool

	Stop() error
}
