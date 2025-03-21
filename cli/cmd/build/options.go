// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package build

var opts = &options{}

type options struct {
	ConfigFile string
}

func init() {
	flags := Command.Flags()
	flags.StringVarP(&opts.ConfigFile, "config", "c", "", "Path to the build configuration file. Supported formats: YAML")
}
