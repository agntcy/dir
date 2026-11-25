// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package daemon

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	runtimestoresql "github.com/agntcy/dir-runtime/store/sql"
	runtimeutils "github.com/agntcy/dir-runtime/utils"
)

const (
	runtimeInstanceIDFile = "runtime-instance-id"
	runtimeInstanceIDPerm = 0o600
)

// EffectiveInstanceID keeps an SQLite store under the daemon's data directory
// compatible with existing unscoped workloads. Shared stores need a persistent
// ID so each daemon recognizes only the workloads it owns.
func (c RuntimeConfig) EffectiveInstanceID(dataDir string) (string, error) {
	if c.InstanceID != "" {
		if strings.TrimSpace(c.InstanceID) == "" {
			return "", errors.New("runtime.instance_id must not be blank")
		}

		return c.InstanceID, nil
	}

	if c.Store.Type == runtimestoresql.StoreTypeSqlite {
		local, err := localRuntimeSQLiteStore(c.Store.SQLite.Path, dataDir)
		if err != nil {
			return "", err
		}

		if local {
			return "", nil
		}
	}

	path := filepath.Join(dataDir, runtimeInstanceIDFile)
	if data, err := os.ReadFile(path); err == nil {
		id := strings.TrimSpace(string(data))
		if id == "" {
			return "", fmt.Errorf("runtime instance ID file %s is empty", path)
		}

		return id, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("failed to read runtime instance ID file: %w", err)
	}

	var random [16]byte
	if _, err := rand.Read(random[:]); err != nil {
		return "", fmt.Errorf("failed to generate runtime instance ID: %w", err)
	}

	id := hex.EncodeToString(random[:])

	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, runtimeInstanceIDPerm)
	if errors.Is(err, os.ErrExist) {
		// Another daemon created the ID after the first read; use its value.
		return c.EffectiveInstanceID(dataDir)
	}
	if err != nil {
		return "", fmt.Errorf("failed to create runtime instance ID file: %w", err)
	}

	if _, err := fmt.Fprintln(file, id); err != nil {
		_ = file.Close()
		_ = os.Remove(path)

		return "", fmt.Errorf("failed to write runtime instance ID file: %w", err)
	}

	if err := file.Close(); err != nil {
		_ = os.Remove(path)

		return "", fmt.Errorf("failed to close runtime instance ID file: %w", err)
	}

	return id, nil
}

func localRuntimeSQLiteStore(storePath, dataDir string) (bool, error) {
	if storePath == "" {
		storePath = runtimestoresql.DefaultPath
	}

	// Match dir-runtime's home expansion before comparing actual store locations.
	expanded, err := runtimeutils.ExpandHome(storePath)
	if err != nil {
		return false, fmt.Errorf("failed to expand runtime SQLite path: %w", err)
	}

	storeAbs, err := filepath.Abs(expanded)
	if err != nil {
		return false, fmt.Errorf("failed to resolve runtime SQLite path: %w", err)
	}

	dataAbs, err := filepath.Abs(dataDir)
	if err != nil {
		return false, fmt.Errorf("failed to resolve daemon data directory: %w", err)
	}

	if !strings.EqualFold(filepath.VolumeName(dataAbs), filepath.VolumeName(storeAbs)) {
		return false, nil
	}

	rel, err := filepath.Rel(dataAbs, storeAbs)
	if err != nil {
		return false, fmt.Errorf("failed to compare runtime SQLite and daemon data paths: %w", err)
	}

	return filepath.IsLocal(rel), nil
}
