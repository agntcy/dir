// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package file

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/agntcy/dir/hub/utils/dir"
)

func CreateAll(path string) (*os.File, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return nil, fmt.Errorf("error creating directory: %w", err)
	}

	file, err := os.Create(path)
	if err != nil {
		return nil, fmt.Errorf("error creating file: %w", err)
	}

	return file, nil
}

func GetSessionFilePath() string {
	return filepath.Join(dir.GetAppDir(), "session.json")
}
