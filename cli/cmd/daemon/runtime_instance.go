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
)

const runtimeInstanceIDFile = "runtime-instance-id"

// EffectiveInstanceID keeps the existing unscoped local SQLite store compatible.
// Shared stores need an ID that survives daemon restarts so their workloads can
// be recognized and reaped by the same discovery instance.
func (c RuntimeConfig) EffectiveInstanceID(dataDir string) (string, error) {
	if c.InstanceID != "" {
		if strings.TrimSpace(c.InstanceID) == "" {
			return "", errors.New("runtime.instance_id must not be blank")
		}

		return c.InstanceID, nil
	}

	if c.Store.Type == runtimestoresql.StoreTypeSqlite {
		return "", nil
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
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
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
