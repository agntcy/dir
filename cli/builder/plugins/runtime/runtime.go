// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package runtime

import (
	"context"
	"fmt"

	objectsv1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/objects/v1"
	coretypes "github.com/agntcy/dir/api/core/v1alpha1"
	"github.com/agntcy/dir/cli/builder/plugins/runtime/framework"
	"github.com/agntcy/dir/cli/builder/plugins/runtime/language"
	"github.com/agntcy/dir/cli/types"
)

type runtime struct {
	framework *framework.Framework
	language  *language.Language
}

func New(source string) types.Builder {
	return &runtime{
		framework: framework.New(source),
		language:  language.New(source),
	}
}

func (c *runtime) Build(ctx context.Context) (*coretypes.Agent, error) {
	frameworkExtension, err := c.framework.Build(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get framework: %w", err)
	}

	languageExtension, err := c.language.Build(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get language: %w", err)
	}

	return &coretypes.Agent{
		Agent: &objectsv1.Agent{
			Extensions: []*objectsv1.Extension{
				frameworkExtension,
				languageExtension,
			},
		},
	}, nil
}
