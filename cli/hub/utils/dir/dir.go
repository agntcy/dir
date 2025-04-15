// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package dir

import (
	"os"
	"path/filepath"
	"runtime"
)

const (
	OsMac     = "darwin"
	OsWindows = "windows"
	OsLinux   = "linux"

	EnvWindowsHomeDir = "USERPROFILE"
	EnvMacHome        = "HOME"
	EnvLinuxHome      = "HOME"
)

func GetHomeDir() string {
	switch runtime.GOOS {
	case OsWindows:
		return os.Getenv(EnvWindowsHomeDir)
	case OsLinux:
		return os.Getenv(EnvLinuxHome)
	case OsMac:
		return os.Getenv(EnvMacHome)
	}

	return ""
}

func GetAppDir() string {
	return filepath.Join(GetHomeDir(), ".dirctl")
}
