// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"testing"
	"time"

	policyv1 "github.com/agntcy/dir/api/policy/v1"
	"github.com/agntcy/dir/server/authn"
	"github.com/agntcy/dir/server/types"
	"github.com/spiffe/go-spiffe/v2/spiffeid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const auditor = "spiffe://example.org/ns/dir/sa/auditor"

var auditPolicy = types.EnforcedPolicy{ID: "opa:a", Version: "v1"}

// auditVerdict is a stored verdict.
type auditVerdict struct {
	types.PolicyEvaluationObject

	compliant bool
}

func (v auditVerdict) GetPolicyID() string      { return auditPolicy.ID }
func (v auditVerdict) GetPolicyVersion() string { return auditPolicy.Version }
func (v auditVerdict) GetStatus() string        { return types.PolicyEvalStatusEvaluated }
func (v auditVerdict) GetCompliant() bool       { return v.compliant }
func (v auditVerdict) GetReason() string        { return "license missing" }
func (v auditVerdict) GetUpdatedAt() time.Time  { return time.Unix(1700000000, 0) }

// auditDB holds gatedCID's compliance, and the records it lists as excluded.
type auditDB struct {
	compliant bool
	excluded  []string
	err       error

	asked [][]types.EnforcedPolicy
	limit int
	skip  int
}

func (d *auditDB) GetPolicyEvaluations(string) ([]types.PolicyEvaluationObject, error) {
	return []types.PolicyEvaluationObject{auditVerdict{compliant: d.compliant}}, d.err
}

func (d *auditDB) IsRecordCompliant(_ string, policies []types.EnforcedPolicy) (bool, error) {
	d.asked = append(d.asked, policies)

	return d.compliant, d.err
}

func (d *auditDB) ListRecordsExcluded(policies []types.EnforcedPolicy, limit, offset int) ([]string, error) {
	d.asked = append(d.asked, policies)
	d.limit, d.skip = limit, offset

	return d.excluded, d.err
}

// auditLog keeps every audit line.
type auditLog struct {
	mu    sync.Mutex
	lines []map[string]any
}

func (l *auditLog) Enabled(context.Context, slog.Level) bool { return true }
func (l *auditLog) WithAttrs([]slog.Attr) slog.Handler       { return l }
func (l *auditLog) WithGroup(string) slog.Handler            { return l }

func (l *auditLog) Handle(_ context.Context, record slog.Record) error {
	line := map[string]any{"msg": record.Message}

	record.Attrs(func(attr slog.Attr) bool {
		line[attr.Key] = attr.Value.Any()

		return true
	})

	l.mu.Lock()
	defer l.mu.Unlock()

	l.lines = append(l.lines, line)

	return nil
}

func (l *auditLog) all() []map[string]any {
	l.mu.Lock()
	defer l.mu.Unlock()

	return l.lines
}

func newAuditController(db *auditDB, store types.StoreAPI, policies ...types.EnforcedPolicy) (*policyAuditCtlr, *auditLog) {
	log := &auditLog{}

	return &policyAuditCtlr{
		db:          db,
		store:       store,
		enforcement: func() types.PolicyEnforcement { return types.PolicyEnforcement{Policies: policies} },
		audit:       slog.New(log),
	}, log
}

func asAuditor(t *testing.T) context.Context {
	t.Helper()

	return context.WithValue(t.Context(), authn.SpiffeIDContextKey, spiffeid.RequireFromString(auditor))
}

// An auditor gets a record whatever the policies say of it, with why, and the
// read is logged with who made it.
func TestPolicyAudit_GetRecord(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		compliant bool
		policies  []types.EnforcedPolicy
		excluded  bool
	}{
		{"excluded record", false, []types.EnforcedPolicy{auditPolicy}, true},
		{"compliant record", true, []types.EnforcedPolicy{auditPolicy}, false},
		{"no policy in force", false, nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db := &auditDB{compliant: tt.compliant}
			ctlr, log := newAuditController(db, &gatedStore{present: true}, tt.policies...)

			resp, err := ctlr.GetRecord(asAuditor(t), &policyv1.GetRecordRequest{Cid: gatedCID})
			require.NoError(t, err)

			assert.NotNil(t, resp.GetRecord())
			assert.Equal(t, tt.excluded, resp.GetExcluded())
			require.Len(t, resp.GetEvaluations(), 1)
			assert.Equal(t, "opa:a", resp.GetEvaluations()[0].GetPolicyId())
			assert.Equal(t, tt.compliant, resp.GetEvaluations()[0].GetCompliant())
			assert.Equal(t, "license missing", resp.GetEvaluations()[0].GetReason())

			assert.Equal(t, []map[string]any{{
				"msg": "Policy audit read", "method": "GetRecord", "caller": auditor, "cid": gatedCID, "excluded": tt.excluded,
			}}, log.all())
		})
	}
}

// Failed reads are logged too, and leak nothing internal.
func TestPolicyAudit_GetRecordFailures(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		cid   string
		store *gatedStore
		db    *auditDB
		code  codes.Code
	}{
		{"invalid CID", "not-a-cid", &gatedStore{present: true}, &auditDB{}, codes.InvalidArgument},
		{"record not held", gatedCID, &gatedStore{present: false}, &auditDB{}, codes.NotFound},
		{"database error", gatedCID, &gatedStore{present: true}, &auditDB{err: errors.New("connection refused")}, codes.Internal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctlr, log := newAuditController(tt.db, tt.store, auditPolicy)

			_, err := ctlr.GetRecord(asAuditor(t), &policyv1.GetRecordRequest{Cid: tt.cid})
			require.Error(t, err)
			assert.Equal(t, tt.code, status.Code(err))
			assert.NotContains(t, err.Error(), "connection refused")

			var reads []map[string]any

			for _, line := range log.all() {
				if line["method"] == "GetRecord" {
					reads = append(reads, line)
				}
			}

			require.Len(t, reads, 1)
			assert.Equal(t, "Policy audit read failed", reads[0]["msg"])
			assert.Equal(t, auditor, reads[0]["caller"])
			assert.Equal(t, tt.cid, reads[0]["cid"])
		})
	}
}

// auditStream collects what ListExcludedRecords sends.
type auditStream struct {
	policyv1.PolicyAuditService_ListExcludedRecordsServer

	ctx     context.Context //nolint:containedctx // a gRPC stream carries its context
	sent    []*policyv1.ListExcludedRecordsResponse
	failAt  int
	sendErr error
}

func (s *auditStream) Context() context.Context { return s.ctx }

func (s *auditStream) Send(resp *policyv1.ListExcludedRecordsResponse) error {
	if s.sendErr != nil && len(s.sent) == s.failAt {
		return s.sendErr
	}

	s.sent = append(s.sent, resp)

	return nil
}

func streamedCIDs(stream *auditStream) []string {
	cids := make([]string, 0, len(stream.sent))
	for _, resp := range stream.sent {
		cids = append(cids, resp.GetCid())
	}

	return cids
}

// The listing streams the excluded records under the policies in force,
// pages as asked, and logs which records it handed out.
func TestPolicyAudit_ListExcludedRecords(t *testing.T) {
	t.Parallel()

	db := &auditDB{excluded: []string{"cid-a", "cid-b"}}
	ctlr, log := newAuditController(db, &gatedStore{}, auditPolicy)
	stream := &auditStream{ctx: asAuditor(t)}

	require.NoError(t, ctlr.ListExcludedRecords(&policyv1.ListExcludedRecordsRequest{Limit: 2, Offset: 4}, stream))

	assert.Equal(t, []string{"cid-a", "cid-b"}, streamedCIDs(stream))
	assert.Len(t, stream.sent[0].GetEvaluations(), 1)
	assert.Equal(t, [][]types.EnforcedPolicy{{auditPolicy}}, db.asked)
	assert.Equal(t, 2, db.limit)
	assert.Equal(t, 4, db.skip)

	assert.Equal(t, []map[string]any{{
		"msg": "Policy audit listing", "method": "ListExcludedRecords", "caller": auditor, "cids": []string{"cid-a", "cid-b"},
	}}, log.all())
}

// With no policy in force nothing is excluded, and the database is not asked.
func TestPolicyAudit_ListWithNothingEnforced(t *testing.T) {
	t.Parallel()

	db := &auditDB{excluded: []string{"cid-a"}}
	ctlr, log := newAuditController(db, &gatedStore{})
	stream := &auditStream{ctx: asAuditor(t)}

	require.NoError(t, ctlr.ListExcludedRecords(&policyv1.ListExcludedRecordsRequest{}, stream))

	assert.Empty(t, stream.sent)
	assert.Empty(t, db.asked)
	require.Len(t, log.all(), 1)
}

// When streaming stops part way, the log says which records went out.
func TestPolicyAudit_ListLogsWhatWasSentBeforeAFailure(t *testing.T) {
	t.Parallel()

	db := &auditDB{excluded: []string{"cid-a", "cid-b"}}
	ctlr, log := newAuditController(db, &gatedStore{}, auditPolicy)
	stream := &auditStream{ctx: asAuditor(t), failAt: 1, sendErr: errors.New("client gone")}

	require.Error(t, ctlr.ListExcludedRecords(&policyv1.ListExcludedRecordsRequest{}, stream))

	lines := log.all()
	require.Len(t, lines, 1)
	assert.Equal(t, "Policy audit listing failed", lines[0]["msg"])
	assert.Equal(t, []string{"cid-a"}, lines[0]["cids"])
}
