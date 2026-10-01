// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Every error is a gRPC status for the caller, or the store's own, returned
// as is.
//
//nolint:wrapcheck
package controller

import (
	"context"
	"log/slog"

	corev1 "github.com/agntcy/dir/api/core/v1"
	policyv1 "github.com/agntcy/dir/api/policy/v1"
	"github.com/agntcy/dir/server/authn"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/utils/logging"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// auditLogger records every read that bypasses the content-policy gate.
var auditLogger = logging.Logger("audit")

// PolicyAuditDatabase is what the policy audit service reads: verdicts, and
// compliance with the policies in force, whatever the reads it serves enforce.
type PolicyAuditDatabase interface {
	GetPolicyEvaluations(recordCID string) ([]types.PolicyEvaluationObject, error)
	IsRecordCompliant(cid string, policies []types.EnforcedPolicy) (bool, error)
	ListRecordsExcluded(policies []types.EnforcedPolicy, limit, offset int) ([]string, error)
}

type policyAuditCtlr struct {
	db          PolicyAuditDatabase
	store       types.StoreAPI
	enforcement func() types.PolicyEnforcement
	audit       *slog.Logger
}

// NewPolicyAuditController serves records the enforced content policies
// exclude. db and store must be the unfiltered ones: this is the one read
// path the policy gate does not apply to, so every call is audit-logged.
func NewPolicyAuditController(db PolicyAuditDatabase, store types.StoreAPI, enforcement func() types.PolicyEnforcement) policyv1.PolicyAuditServiceServer {
	return &policyAuditCtlr{db: db, store: store, enforcement: enforcement, audit: auditLogger}
}

func (c *policyAuditCtlr) GetRecord(ctx context.Context, req *policyv1.GetRecordRequest) (*policyv1.GetRecordResponse, error) {
	cid := req.GetCid()

	resp, err := c.getRecord(ctx, cid)
	if err != nil {
		c.log(ctx, "GetRecord", "Policy audit read failed", "cid", cid, "error", err)

		return nil, err
	}

	c.log(ctx, "GetRecord", "Policy audit read", "cid", cid, "excluded", resp.GetExcluded())

	return resp, nil
}

func (c *policyAuditCtlr) getRecord(ctx context.Context, cid string) (*policyv1.GetRecordResponse, error) {
	if _, err := corev1.ConvertCIDToDigest(cid); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid record CID %q", cid)
	}

	record, err := c.store.Pull(ctx, &corev1.RecordRef{Cid: cid})
	if err != nil {
		return nil, err
	}

	excluded := false

	if policies := c.enforcement().Policies; len(policies) > 0 {
		compliant, err := c.db.IsRecordCompliant(cid, policies)
		if err != nil {
			return nil, c.internal("check record against enforced policies", err)
		}

		excluded = !compliant
	}

	evaluations, err := c.evaluations(cid)
	if err != nil {
		return nil, err
	}

	return &policyv1.GetRecordResponse{Record: record, Excluded: excluded, Evaluations: evaluations}, nil
}

func (c *policyAuditCtlr) ListExcludedRecords(req *policyv1.ListExcludedRecordsRequest, srv policyv1.PolicyAuditService_ListExcludedRecordsServer) error {
	ctx := srv.Context()

	sent, err := c.listExcludedRecords(req, srv)
	if err != nil {
		c.log(ctx, "ListExcludedRecords", "Policy audit listing failed", "cids", sent, "error", err)

		return err
	}

	c.log(ctx, "ListExcludedRecords", "Policy audit listing", "cids", sent)

	return nil
}

// listExcludedRecords streams the excluded records, returning the CIDs it
// sent.
func (c *policyAuditCtlr) listExcludedRecords(req *policyv1.ListExcludedRecordsRequest, srv policyv1.PolicyAuditService_ListExcludedRecordsServer) ([]string, error) {
	policies := c.enforcement().Policies
	if len(policies) == 0 {
		return nil, nil
	}

	cids, err := c.db.ListRecordsExcluded(policies, int(req.GetLimit()), int(req.GetOffset()))
	if err != nil {
		return nil, c.internal("list records excluded by enforced policies", err)
	}

	sent := make([]string, 0, len(cids))

	for _, cid := range cids {
		evaluations, err := c.evaluations(cid)
		if err != nil {
			return sent, err
		}

		if err := srv.Send(&policyv1.ListExcludedRecordsResponse{Cid: cid, Evaluations: evaluations}); err != nil {
			return sent, err
		}

		sent = append(sent, cid)
	}

	return sent, nil
}

func (c *policyAuditCtlr) evaluations(cid string) ([]*policyv1.PolicyEvaluation, error) {
	rows, err := c.db.GetPolicyEvaluations(cid)
	if err != nil {
		return nil, c.internal("get policy evaluations", err)
	}

	evaluations := make([]*policyv1.PolicyEvaluation, 0, len(rows))
	for _, row := range rows {
		evaluations = append(evaluations, &policyv1.PolicyEvaluation{
			PolicyId:      row.GetPolicyID(),
			PolicyVersion: row.GetPolicyVersion(),
			Status:        row.GetStatus(),
			Compliant:     row.GetCompliant(),
			Reason:        row.GetReason(),
			UpdatedAt:     timestamppb.New(row.GetUpdatedAt()),
		})
	}

	return evaluations, nil
}

// internal logs err and answers Internal without its detail.
func (c *policyAuditCtlr) internal(what string, err error) error {
	c.audit.Error("Policy audit: failed to "+what, "error", err)

	return status.Error(codes.Internal, "failed to "+what)
}

// log writes one audit line, naming the caller.
func (c *policyAuditCtlr) log(ctx context.Context, method, msg string, attrs ...any) {
	caller := ""

	if id, ok := authn.SpiffeIDFromContext(ctx); ok {
		caller = id.String()
	}

	c.audit.Info(msg, append([]any{"method", method, "caller", caller}, attrs...)...)
}
