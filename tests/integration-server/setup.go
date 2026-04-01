// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package integration_server

import (
	"context"
	"crypto/sha1" //nolint:gosec
	"encoding/hex"
	"fmt"
	"net"

	"github.com/agntcy/dir/server"
	"github.com/agntcy/dir/server/config"
	"github.com/agntcy/dir/server/database" //nolint:typecheck
	gormdb "github.com/agntcy/dir/server/database/gorm"
	"github.com/agntcy/dir/server/events"
	"github.com/agntcy/dir/server/store/oci"
	"github.com/agntcy/dir/server/types"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
	"gorm.io/gorm"
	"oras.land/oras-go/v2/registry/remote"
)

const (
	bufSize = 1024 * 1024
)

type TestVars struct {
	options    types.APIOptions
	repository *remote.Repository
	db         *gorm.DB
	server     *grpc.Server
	conn       *grpc.ClientConn
}

var t TestVars

var _ = ginkgo.BeforeEach(func(ctx ginkgo.SpecContext) {
	t = TestVars{}
	t.options = getOptions()
	t.repository = newRepository(t.options)

	// Wrap the spec in a database transaction.
	db := newDb(t.options)
	tx := db.Begin()
	t.db = tx

	t.server, t.conn = startServer(ctx, t.options, t.db)
})

var _ = ginkgo.AfterEach(func(ctx ginkgo.SpecContext) {
	errors := []error{}

	err := t.conn.Close()
	if err != nil {
		errors = append(errors, err)
	}

	t.server.Stop()

	t.db.Rollback()

	if t.db.Error != nil {
		errors = append(errors, t.db.Error)
	}

	err = deleteSpecRepository(ctx)
	if err != nil {
		errors = append(errors, err)
	}

	if 0 < len(errors) {
		ginkgo.Fail(fmt.Sprintf("AfterEach errors: %v", errors))
	}
})

func newRepository(options types.APIOptions) *remote.Repository {
	repository, err := oci.NewORASRepository(options.Config().Store.OCI)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	return repository
}

func newDb(options types.APIOptions) *gorm.DB {
	db, err := database.NewPostgresGormDb(options.Config().Database.Postgres)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	return db
}

// startServer starts a gRPC server in the background for the current spec.
func startServer(ctx context.Context, options types.APIOptions, db *gorm.DB) (*grpc.Server, *grpc.ClientConn) {
	databaseAPI, err := gormdb.New(db)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	s, err := server.New(ctx, options.Config(), databaseAPI)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	grpcServer := s.GrpcServer()
	listener := bufconn.Listen(bufSize)

	go func() {
		if err := grpcServer.Serve(listener); err != nil {
			panic(err)
		}
	}()

	conn, err := grpc.NewClient(
		"passthrough://bufnet",
		grpc.WithContextDialer(bufDialer(listener)),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)

	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	return grpcServer, conn
}

func bufDialer(listener *bufconn.Listener) func(context.Context, string) (net.Conn, error) {
	return func(ctx context.Context, _ string) (net.Conn, error) {
		return listener.DialContext(ctx)
	}
}

// getOptions generates the options for the current spec.
// It's assumed the environment variables are already loaded at this stage.
func getOptions() types.APIOptions {
	c, err := config.LoadConfig()
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	c.Store.OCI.RepositoryName = fmt.Sprintf(
		"%s-%d-%s",
		c.Store.OCI.RepositoryName,
		ginkgo.GinkgoParallelProcess(),
		getCurrentSpecID(),
	)

	options := types.NewOptions(c)
	eventBus := events.NewSafeEventBus(nil)

	return options.WithEventBus(eventBus)
}

// getCurrentSpecID calculates the unique ID of the current spec.
// The ID is an 8 character long hexadecimal string.
func getCurrentSpecID() string {
	report := ginkgo.CurrentSpecReport()
	fullText := report.FullText()
	checksum := sha1.Sum([]byte(fullText)) //nolint:gosec
	encoded := hex.EncodeToString(checksum[:])

	return encoded[:8]
}

// deleteSpecRepository deletes the repository associated with the current spec.
func deleteSpecRepository(ctx context.Context) error {
	// Unfortunately, there's no ORAS operation or Zot endpoint to delete a repository.
	// So, the only way to delete a repository is to delete its contents.
	return t.repository.Tags(ctx, "", func(tags []string) error { //nolint:wrapcheck
		for _, tag := range tags {
			desc, err := t.repository.Resolve(ctx, tag)
			if err != nil {
				return fmt.Errorf("failed to resolve tag %s: %w", tag, err)
			}

			err = t.repository.Delete(ctx, desc)
			if err != nil {
				return fmt.Errorf("failed to delete content %s: %w", tag, err)
			}
		}

		return nil
	})
}
