package runtime

import (
	"github.com/agntcy/dir/server/runtime/docker"
	"github.com/agntcy/dir/server/types"
)

func New(store types.StoreAPI) (types.DiscoveryAPI, error) {
	return docker.New(store)
}
