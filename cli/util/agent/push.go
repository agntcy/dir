// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package agent

import (
	"errors"
	"fmt"
	"io"
	"os"
)

func GetReader(fpath string, fromFile bool) (io.ReadCloser, error) {
	if fpath == "" && !fromFile {
		return nil, errors.New("if no path defined --stdin flag must be set")
	}

	if fpath != "" {
		file, err := os.Open(fpath)
		if err != nil {
			return nil, fmt.Errorf("could not open file %s: %w", fpath, err)
		}

		return file, nil
	}

	return os.Stdin, nil
}

func GetAgentBytes(reader io.Reader) ([]byte, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read data: %w", err)
	}

	return data, nil
}
