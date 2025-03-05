// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package build

var opts = &options{}

type options struct {
	ConfigFile string
}

func init() {
	flags := Command.Flags()
	flags.StringVarP(&opts.ConfigFile, "config-file", "f", "", "Path to the agent build configuration file. Please note that other flags will override the build configuration from the file. Supported formats: YAML")
}
