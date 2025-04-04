// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package options

import "github.com/spf13/cobra"

type BuildOptions struct {
	*BaseOption

	ConfigFile string
}

func NewBuildOptions(base *BaseOption, cmd *cobra.Command) *BuildOptions {
	opts := &BuildOptions{
		BaseOption: base,
	}

	opts.AddRegisterFns(func() error {
		flags := cmd.Flags()
		flags.StringVarP(&opts.ConfigFile, "config", "c", "", "Path to the build configuration file. Supported formats: YAML")

		return nil
	})

	return &BuildOptions{
		BaseOption: base,
	}
}
