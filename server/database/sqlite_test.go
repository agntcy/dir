// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package database

import (
	"path/filepath"
	"sync"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestWithForeignKeysBuildsADSNPerPathShape(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"/var/lib/dir/dir.db":           "/var/lib/dir/dir.db?_pragma=foreign_keys(1)",
		":memory:":                      ":memory:?_pragma=foreign_keys(1)",
		"file::memory:?cache=shared":    "file::memory:?cache=shared&_pragma=foreign_keys(1)",
		"file:dir.db?_txlock=immediate": "file:dir.db?_txlock=immediate&_pragma=foreign_keys(1)",
	}

	for path, want := range tests {
		assert.Equal(t, want, withForeignKeys(path), "path %q", path)
	}
}

// Foreign keys are a per-connection setting in SQLite. Running the pragma as a
// statement after Open only reaches one connection, leaving the rest of the
// pool to silently skip the ON DELETE CASCADE on Record's associations.
func TestForeignKeysAreOnForEveryPooledConnection(t *testing.T) {
	t.Parallel()

	for name, path := range map[string]string{
		"file":   filepath.Join(t.TempDir(), "pool.db"),
		"memory": "file::memory:?cache=shared",
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			db, err := gorm.Open(sqlite.Open(withForeignKeys(path)), &gorm.Config{})
			require.NoError(t, err)

			require.NoError(t, configureSQLPool(db))

			sqlDB, err := db.DB()
			require.NoError(t, err)

			// Hold every connection open at once so the pool has to open new
			// ones rather than handing back the one Open configured.
			const conns = 8

			var (
				wg      sync.WaitGroup
				ready   sync.WaitGroup
				mu      sync.Mutex
				enabled []int
				release = make(chan struct{})
			)

			ready.Add(conns)
			wg.Add(conns)

			for range conns {
				go func() {
					defer wg.Done()

					conn, err := sqlDB.Conn(t.Context())
					if !assert.NoError(t, err) {
						ready.Done()

						return
					}
					defer conn.Close()

					var fk int
					if assert.NoError(t, conn.QueryRowContext(t.Context(), "PRAGMA foreign_keys").Scan(&fk)) {
						mu.Lock()

						enabled = append(enabled, fk)

						mu.Unlock()
					}

					ready.Done()
					<-release
				}()
			}

			ready.Wait()
			close(release)
			wg.Wait()

			require.Len(t, enabled, conns)

			for _, fk := range enabled {
				assert.Equal(t, 1, fk, "a pooled connection had foreign keys off, so cascade deletes are skipped on it")
			}
		})
	}
}
