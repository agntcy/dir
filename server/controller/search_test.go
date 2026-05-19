// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"errors"
	"testing"
	"time"

	corev1 "github.com/agntcy/dir/api/core/v1"
	searchv1 "github.com/agntcy/dir/api/search/v1"
	rankingcfg "github.com/agntcy/dir/server/ranking/config"
	"github.com/agntcy/dir/server/types"
	"google.golang.org/grpc"
)

// TestSearchCIDsDedupesByCID pins the JOIN-amplification dedup in
// fetchAndScore: a record with multiple matching child rows must
// surface as one ranked response, not N. The e2e suite can't reliably
// exercise this because it depends on fixture association mix.
func TestSearchCIDsDedupesByCID(t *testing.T) {
	t.Parallel()

	rec := newFakeRecord("cid-dup", fakeOpts{
		signed:        true,
		schemaVersion: "1.0.0",
		createdAt:     time.Now().UTC().Format(time.RFC3339),
		skills:        []string{"a", "b"},
	})

	db := &fakeSearchDB{records: []types.Record{rec, rec, rec}}
	ctlr := NewSearchController(db, nil, rankingcfg.Config{
		MaxCandidates: rankingcfg.DefaultMaxCandidates,
	})

	srv := &fakeSearchCIDsServer{ctx: context.Background()}
	if err := ctlr.SearchCIDs(&searchv1.SearchCIDsRequest{}, srv); err != nil {
		t.Fatalf("SearchCIDs returned error: %v", err)
	}

	if got, want := len(srv.sent), 1; got != want {
		t.Fatalf("got %d responses, want %d (controller must dedupe by CID)",
			got, want)
	}

	if got, want := srv.sent[0].GetRecordCid(), "cid-dup"; got != want {
		t.Errorf("got cid %q, want %q", got, want)
	}
}

// --- Test doubles ----------------------------------------------------------

type fakeSearchDB struct {
	types.DatabaseAPI

	records []types.Record
}

func (f *fakeSearchDB) GetRecords(_ ...types.FilterOption) ([]types.Record, error) {
	return f.records, nil
}

// The ranking layer calls these during scoring; stubs prevent the
// embedded nil DatabaseAPI from panicking.
func (f *fakeSearchDB) GetSignatureVerificationsByRecordCID(_ string) ([]types.SignatureVerificationObject, error) {
	return nil, nil
}

func (f *fakeSearchDB) GetVerificationByCID(_ string) (types.NameVerificationObject, error) {
	return nil, errors.New("not found")
}

type fakeRecord struct {
	cid    string
	signed bool
	data   *fakeRecordData
}

type fakeOpts struct {
	signed        bool
	schemaVersion string
	createdAt     string
	skills        []string
}

func newFakeRecord(cid string, opts fakeOpts) *fakeRecord {
	return &fakeRecord{
		cid:    cid,
		signed: opts.signed,
		data: &fakeRecordData{
			schemaVersion: opts.schemaVersion,
			createdAt:     opts.createdAt,
			skills:        opts.skills,
		},
	}
}

func (f *fakeRecord) GetCid() string                           { return f.cid }
func (f *fakeRecord) GetRecordData() (types.RecordData, error) { return f.data, nil }
func (f *fakeRecord) GetSigned() bool                          { return f.signed }

type fakeRecordData struct {
	types.RecordData

	schemaVersion string
	createdAt     string
	skills        []string
}

func (f *fakeRecordData) GetSchemaVersion() string      { return f.schemaVersion }
func (f *fakeRecordData) GetCreatedAt() string          { return f.createdAt }
func (f *fakeRecordData) GetSignature() types.Signature { return nil }

func (f *fakeRecordData) GetSkills() []types.Skill {
	out := make([]types.Skill, len(f.skills))
	for i, s := range f.skills {
		out[i] = &fakeSkill{name: s}
	}

	return out
}

func (f *fakeRecordData) GetDomains() []types.Domain   { return nil }
func (f *fakeRecordData) GetModules() []types.Module   { return nil }
func (f *fakeRecordData) GetLocators() []types.Locator { return nil }

type fakeSkill struct {
	types.Skill

	name string
}

func (f *fakeSkill) GetName() string                   { return f.name }
func (f *fakeSkill) GetID() uint64                     { return 0 }
func (f *fakeSkill) GetAnnotations() map[string]string { return nil }

type fakeSearchCIDsServer struct {
	grpc.ServerStream

	ctx  context.Context //nolint:containedctx // mock gRPC stream
	sent []*searchv1.SearchCIDsResponse
}

func (f *fakeSearchCIDsServer) Context() context.Context { return f.ctx }
func (f *fakeSearchCIDsServer) Send(resp *searchv1.SearchCIDsResponse) error {
	f.sent = append(f.sent, resp)

	return nil
}

var _ = corev1.RecordRef{}
