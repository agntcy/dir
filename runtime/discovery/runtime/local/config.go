// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package local

import (
	"github.com/agntcy/dir/runtime/discovery/types"
	"github.com/agntcy/dir/runtime/utils"
)

const RuntimeType types.RuntimeType = "local"

const DefaultEnvSelector = "__AGNTCY_DISCOVERY__=true"

var logger = utils.NewLogger("runtime", string(RuntimeType))

type Config struct {
	EnvSelector string
}
