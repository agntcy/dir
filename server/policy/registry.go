// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package policy

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"slices"

	corev1 "github.com/agntcy/dir/api/core/v1"
	policyv1 "github.com/agntcy/dir/api/policy/v1"
	policyconfig "github.com/agntcy/dir/server/policy/config"
)

// versionLength is how many hex digits of the content hash make a policy
// version: enough that two different policies do not share one, short enough to
// read in a metric label.
const versionLength = 16

// Registry maps operations to the operation policies that decide at them, and
// holds the evaluation policies.
type Registry struct {
	byOp        map[string][]operationPolicy
	evaluations []evaluation

	// verdicts and enforcement are what a policy deciding on stored verdicts
	// needs and the registry cannot work out for itself. Both are nil in a
	// process that serves no record by CID, which is every process but the
	// apiserver; a policy that would need them is refused at load time there
	// too, so nothing reaches a decision expecting them.
	verdicts    VerdictSource
	enforcement EnforcementSource
}

// RegistryOption supplies what a registry needs beyond its configuration.
type RegistryOption func(*Registry)

// WithVerdicts gives the registry the stored verdicts an operation policy can
// decide on, and the enforcement that says which version of each policy is in
// force.
//
// The two arrive together because neither is usable alone: a verdict without
// the version in force cannot be told from a conclusion about a rule since
// edited, which is the one distinction the gate depends on.
func WithVerdicts(source VerdictSource, enforcement EnforcementSource) RegistryOption {
	return func(r *Registry) {
		r.verdicts = source
		r.enforcement = enforcement
	}
}

// operationPolicy is one loaded operation policy: what decides, what it is
// called, and how its decision applies.
//
// The mode is kept here because it is a property of the configured policy, not
// of the decision: the same validator in two entries can enforce at one
// operation and only report at another.
type operationPolicy struct {
	id        string
	mode      policyconfig.Mode
	validator Validator

	// verdicts is what this policy's source says about the verdicts it reads.
	verdicts verdictRefs
}

// compiled is what loading one configured provider produced: the validator
// itself and everything that could only be read while its source was in hand.
type compiled struct {
	validator Validator

	// version is the hash of the policy's content, empty for a provider whose
	// content is not here to hash.
	version string

	// factGroups are the metadata groups the policy reads.
	factGroups []string

	// verdicts is what the policy says about the verdicts it reads.
	verdicts verdictRefs

	// readsRecord is whether the policy reads the record's content.
	readsRecord bool
}

// evaluation is one loaded evaluation policy. Evaluators turns it into an
// Evaluator once it has a source of record content.
type evaluation struct {
	id         string
	version    string
	validator  Validator
	factGroups []string
}

// contentVersion is the version of a policy made of parts, in order. Each part
// is length-prefixed so that moving a boundary between two parts changes it.
func contentVersion(parts ...string) string {
	h := sha256.New()

	for _, part := range parts {
		fmt.Fprintf(h, "%d:%s", len(part), part)
	}

	return hex.EncodeToString(h.Sum(nil))[:versionLength]
}

// NewRegistry constructs the policies of both blocks. policyDir is the
// directory named .rego files are loaded from (server/reconciler policy.dir).
// Two empty lists yield a registry with no evaluation policies and no
// operation policy for any op.
func NewRegistry(
	ctx context.Context,
	evaluations policyconfig.Evaluations,
	operations policyconfig.Operations,
	policyDir string,
	opts ...RegistryOption,
) (*Registry, error) {
	if err := evaluations.Validate(); err != nil {
		return nil, fmt.Errorf("invalid policy config: %w", err)
	}

	if err := operations.Validate(); err != nil {
		return nil, fmt.Errorf("invalid policy config: %w", err)
	}

	if err := policyconfig.ValidateNames(evaluations, operations); err != nil {
		return nil, fmt.Errorf("invalid policy config: %w", err)
	}

	var loaded []evaluation

	for i, e := range evaluations {
		if e.Provider == policyconfig.ProviderOASF {
			return nil, fmt.Errorf("evaluations[%d]: provider %q cannot define an evaluation policy", i, e.Provider)
		}

		c, err := newValidator(ctx, layerEvaluation, e.Provider,
			provider{file: e.File, expressions: e.Expressions}, policyDir)
		if err != nil {
			return nil, fmt.Errorf("evaluations[%d]: %w", i, err)
		}

		loaded = append(loaded, evaluation{
			id:         e.PolicyID(),
			version:    c.version,
			validator:  c.validator,
			factGroups: c.factGroups,
		})
	}

	byOp := make(map[string][]operationPolicy)

	for i, o := range operations {
		// An operation policy stores no verdict, so nothing caches what it
		// decided and nothing has to be invalidated: its fact groups are not
		// asked for. The verdicts it reads are, because they decide when it
		// may start enforcing.
		c, err := newValidator(ctx, layerOperation, o.Provider,
			provider{schemaURL: o.SchemaURL, file: o.File, expressions: o.Expressions}, policyDir)
		if err != nil {
			return nil, fmt.Errorf("operations[%d]: %w", i, err)
		}

		if err := checkVerdictsAreReadable(o.Ops, c.verdicts); err != nil {
			return nil, fmt.Errorf("operations[%d]: %w", i, err)
		}

		if err := checkRecordIsReadable(o.Ops, c.readsRecord); err != nil {
			return nil, fmt.Errorf("operations[%d]: %w", i, err)
		}

		// Resolved once, here, so that a decision and the log of it report the
		// mode the policy runs in rather than the mode the operator omitted.
		mode := o.Mode
		if mode == "" {
			mode = policyconfig.ModeEnforce
		}

		for _, op := range o.Ops {
			byOp[op] = append(byOp[op], operationPolicy{
				id:        o.PolicyID(),
				mode:      mode,
				validator: c.validator,
				verdicts:  c.verdicts,
			})
		}
	}

	registry := &Registry{byOp: byOp, evaluations: loaded}
	for _, opt := range opts {
		opt(registry)
	}

	if err := registry.checkVerdictsAreAvailable(); err != nil {
		return nil, err
	}

	return registry, nil
}

// checkVerdictsAreAvailable refuses a policy that reads verdicts in a process
// that cannot supply them.
//
// The op check above says verdicts exist at pull; this says this process can
// reach them. They fail differently: the first is a mistake in the config and
// is wrong everywhere, the second is the same config loaded somewhere it does
// not fit, such as the reconciler. Starting anyway would mean a policy
// deciding every record on an empty map.
func (r *Registry) checkVerdictsAreAvailable() error {
	if r.verdicts != nil && r.enforcement != nil {
		return nil
	}

	for op, policies := range r.byOp {
		for _, p := range policies {
			if p.verdicts.reads() {
				return fmt.Errorf("policy %q at op %q reads verdicts, which this process has no source of", p.id, op)
			}
		}
	}

	return nil
}

// checkVerdictsAreReadable rejects a policy that reads verdicts at an
// operation where there are none to read.
//
// Only pull decides about a record the catalogue already holds, so only pull
// has verdicts. At push the record is arriving and no evaluation has seen it;
// at index it is being made searchable, which is what has to happen before an
// evaluation can run; at autosync it is someone else's record arriving.
// Reading verdicts at any of those would be reading an empty map, and the
// policy would refuse everything while appearing to decide.
//
// Rejected here, by name, rather than by leaving verdicts out of some
// environments: a policy may be configured at several operations at once, so
// there is no one environment to compile it in, and a resolution failure
// would say only that the variable does not exist.
func checkVerdictsAreReadable(ops []string, refs verdictRefs) error {
	if !refs.reads() {
		return nil
	}

	for _, op := range ops {
		if op != policyconfig.OpPull {
			return fmt.Errorf("reads verdicts, which only op %q has: op %q decides before any verdict exists",
				policyconfig.OpPull, op)
		}
	}

	return nil
}

// checkRecordIsReadable rejects a policy that reads the record's content at
// pull, where the decision is reached before the record is.
//
// A read by CID is gated before the content is fetched, which is what keeps a
// lookup an index read. Handing such a policy an empty record would let it
// decide on nothing while appearing to decide on content.
//
// The restriction costs less than it looks. A record's content cannot change
// under its CID, so a question about content has one answer forever and
// belongs in an evaluation policy, which sees the whole record, is answered
// once, and leaves a verdict this can read.
func checkRecordIsReadable(ops []string, readsRecord bool) error {
	if !readsRecord {
		return nil
	}

	if slices.Contains(ops, policyconfig.OpPull) {
		return fmt.Errorf("reads the record at op %q, which decides before the record is read: "+
			"put a rule about record content in an evaluation policy and read its verdict here",
			policyconfig.OpPull)
	}

	return nil
}

// provider is the settings of whichever backend is configured. Config
// validation has already rejected the ones that do not apply.
type provider struct {
	schemaURL   string
	file        string
	expressions []string
}

// newValidator builds the validator for one configured provider, the version
// of its content when the provider has any, and the metadata groups its policy
// reads. OASF has neither: it validates a record against a schema, which says
// nothing about what this node observed.
//
// The groups come from here because this is where the policy's source is read.
// Nothing downstream has it: a verdict names a policy and a version, and the
// expressions that reached it are gone by then.
//
// l is the layer the policy was configured in. Only CEL needs it today,
// because it is the only provider that compiles its policy here and so the
// only one that can reject reading the other layer's inputs at load time.
func newValidator(ctx context.Context, l layer, name string, p provider, policyDir string) (compiled, error) {
	switch name {
	case policyconfig.ProviderOASF:
		v, err := newOASFValidator(p.schemaURL)
		if err != nil {
			return compiled{}, fmt.Errorf("failed to initialize OASF validator: %w", err)
		}

		// OASF is nothing but a reading of the record: it checks the content
		// against a schema. There is no sense in which it reads less.
		return compiled{validator: v, readsRecord: true}, nil
	case policyconfig.ProviderCEL:
		v, err := newCELValidator(p.expressions, l)
		if err != nil {
			return compiled{}, fmt.Errorf("failed to initialize CEL validator: %w", err)
		}

		return compiled{
			validator:   v,
			version:     v.version,
			factGroups:  v.factGroups,
			verdicts:    v.verdicts,
			readsRecord: v.readsRecord,
		}, nil
	case policyconfig.ProviderOPA:
		v, err := newOPAValidator(ctx, policyDir, p.file)
		if err != nil {
			return compiled{}, fmt.Errorf("failed to initialize OPA validator: %w", err)
		}

		return compiled{
			validator:   v,
			version:     v.version,
			factGroups:  v.factGroups,
			verdicts:    v.verdicts,
			readsRecord: v.readsRecord,
		}, nil
	default:
		return compiled{}, fmt.Errorf("unsupported provider %q", name)
	}
}

// OperationIDs returns the IDs of the policies configured at op, in config
// order, whatever mode each runs in. The slice is empty if none are.
func (r *Registry) OperationIDs(op string) []string {
	if r == nil {
		return nil
	}

	ids := make([]string, 0, len(r.byOp[op]))
	for _, p := range r.byOp[op] {
		ids = append(ids, p.id)
	}

	return ids
}

// EvaluationIDs returns the evaluation policy IDs, in config order.
func (r *Registry) EvaluationIDs() []string {
	if r == nil {
		return nil
	}

	ids := make([]string, len(r.evaluations))
	for i, e := range r.evaluations {
		ids[i] = e.id
	}

	return ids
}

// Evaluators returns one evaluator per evaluation policy, in config order.
// records supplies the record content a verdict is decided on. facts supplies
// the stored observations projected into meta; a nil source leaves meta unset.
// A nil registry returns no evaluators.
func (r *Registry) Evaluators(records RecordSource, facts FactSource) []Evaluator {
	if r == nil {
		return nil
	}

	out := make([]Evaluator, len(r.evaluations))
	for i, e := range r.evaluations {
		out[i] = &evaluator{
			id:         e.id,
			version:    e.version,
			validator:  e.validator,
			factGroups: e.factGroups,
			records:    records,
			facts:      facts,
		}
	}

	return out
}

// Decide applies every operation policy configured for op, in config order.
// A nil registry or an op with no policies is a no-op and reports that the
// record passes. Messages from every failing enforcing policy are collected.
//
// Each policy applies in its own mode. A policy in off is not run at all. A
// policy in shadow is run and its decision reported, but it does not reach the
// caller: that is the whole of the difference between shadow and enforce, and
// it is why the decision has to be reached per policy rather than once.
//
// Every decision is logged here rather than by the caller, because what was
// decided and by which policy in which mode is known here and nowhere else —
// a shadow decision in particular is never returned at all.
func (r *Registry) Decide(
	ctx context.Context,
	op string,
	record *corev1.Record,
	opts ...DecideOption,
) (bool, []string, error) {
	if r == nil {
		return true, nil, nil
	}

	input := operationInput(op, record, opts...)

	if r.readsVerdicts(op) {
		verdicts, err := r.readVerdicts(record.GetCid())
		if err != nil {
			return false, nil, err
		}

		input.Verdicts = verdicts
	}

	return r.apply(ctx, op, input, func(ctx context.Context, v Validator) (bool, []string, error) {
		return validateOperation(ctx, record, input, v)
	})
}

// DecidePull applies the operation policies at pull to a record known only by
// its CID, which is all a read by CID has: the gate runs before the content
// is fetched, so that a lookup stays an index read.
//
// A policy that would read the record is refused at load time, so what is
// missing here is only what nothing configured can ask for.
func (r *Registry) DecidePull(ctx context.Context, recordCID string) (bool, []string, error) {
	if r == nil || len(r.byOp[policyconfig.OpPull]) == 0 {
		return true, nil, nil
	}

	var verdicts map[string]*policyv1.Verdict

	if r.readsVerdicts(policyconfig.OpPull) {
		read, err := r.readVerdicts(recordCID)
		if err != nil {
			return false, nil, err
		}

		verdicts = read
	}

	input := pullInput(recordCID, verdicts)

	return r.apply(ctx, policyconfig.OpPull, input, func(ctx context.Context, v Validator) (bool, []string, error) {
		return validateOperationByCID(ctx, input, v)
	})
}

// apply runs each policy at op and collects what the enforcing ones refused.
//
// run is how one policy is evaluated, which differs by what the caller has:
// a record to measure and time out against, or only a CID.
func (r *Registry) apply(
	ctx context.Context,
	op string,
	input *policyv1.OperationInput,
	run func(context.Context, Validator) (bool, []string, error),
) (bool, []string, error) {
	var all []string

	for _, p := range r.byOp[op] {
		if p.validator == nil || p.mode == policyconfig.ModeOff {
			continue
		}

		// Resolved per decision, not at load: a policy held back by the
		// latch starts enforcing as soon as what it reads has covered the
		// catalogue, without the server being restarted.
		mode := p.mode
		if mode == policyconfig.ModeEnforce && !r.mayEnforce(p.verdicts) {
			mode = policyconfig.ModeShadow
		}

		ok, msgs, err := run(ctx, p.validator)
		if err != nil {
			return false, nil, fmt.Errorf("decide record with policy %q: %w", p.id, err)
		}

		logDecision(ctx, operationPolicy{id: p.id, mode: mode, validator: p.validator}, input, ok, msgs)

		if !ok && mode == policyconfig.ModeEnforce {
			all = append(all, msgs...)
		}
	}

	return len(all) == 0, all, nil
}

// readsVerdicts reports whether any policy at op decides on a stored verdict,
// which is what makes the read worth doing. An op whose policies read none
// costs no query.
func (r *Registry) readsVerdicts(op string) bool {
	if r.verdicts == nil || r.enforcement == nil {
		return false
	}

	for _, p := range r.byOp[op] {
		if p.mode != policyconfig.ModeOff && p.verdicts.reads() {
			return true
		}
	}

	return false
}

// With returns a registry that runs vs for op, each enforcing, which is what
// an operation policy does when its mode is unset. Intended for tests.
func With(op string, vs ...Validator) *Registry {
	policies := make([]operationPolicy, 0, len(vs))
	for i, v := range vs {
		policies = append(policies, operationPolicy{
			id:        fmt.Sprintf("test:%d", i),
			mode:      policyconfig.ModeEnforce,
			validator: v,
		})
	}

	return &Registry{byOp: map[string][]operationPolicy{op: policies}}
}
