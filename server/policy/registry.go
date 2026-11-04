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
	"github.com/agntcy/dir/server/policy/predicate"
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

	// predicate is what the policy compiles to when it is at search, and nil
	// at every other op, where the policy is evaluated instead.
	predicate predicate.Node
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

	// predicate is what the policy compiles to for a search, and predicateErr
	// why it does not. Exactly one of the two is set, whatever the provider.
	predicate    predicate.Node
	predicateErr error
}

// errNotCompilableProvider is the predicateErr of a provider that compiles
// nothing. Config validation refuses these at search already; this is what
// keeps the compiled invariant total rather than relying on that.
func errNotCompilableProvider(provider string) error {
	return fmt.Errorf("%w: provider %q compiles no query", errNotCompilable, provider)
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

	// Computed before either block is built. Operation policies are checked
	// against this list while they compile, so it has to exist first.
	verdictIDs := evaluationPolicyIDs(evaluations)

	var loaded []evaluation

	for i, e := range evaluations {
		if e.Provider == policyconfig.ProviderOASF {
			return nil, fmt.Errorf("evaluations[%d]: provider %q cannot define an evaluation policy", i, e.Provider)
		}

		c, err := newValidator(ctx, layerEvaluation, e.Provider,
			provider{file: e.File, expressions: e.Expressions}, policyDir, verdictIDs)
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

	byOp, err := loadOperations(ctx, operations, policyDir, verdictIDs)
	if err != nil {
		return nil, err
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

// loadOperations compiles each configured operation policy and indexes it by
// the ops it is configured at, checking that what it reads and what it says
// are available at every one of them.
//
// verdictIDs are the configured evaluation policies a verdict key may name,
// which each expression is checked against as it compiles.
func loadOperations(ctx context.Context, operations policyconfig.Operations, policyDir string, verdictIDs []string,
) (map[string][]operationPolicy, error) {
	byOp := make(map[string][]operationPolicy)

	for i, o := range operations {
		// An operation policy stores no verdict, so nothing caches what it
		// decided and nothing has to be invalidated: its fact groups are not
		// asked for. The verdicts it reads are, because they decide when it
		// may start enforcing.
		c, err := newValidator(ctx, layerOperation, o.Provider,
			provider{schemaURL: o.SchemaURL, file: o.File, expressions: o.Expressions}, policyDir, verdictIDs)
		if err != nil {
			return nil, fmt.Errorf("operations[%d]: policy %q: %w", i, o.PolicyID(), err)
		}

		for _, check := range []error{
			checkVerdictsAreReadable(o.Ops, c.verdicts),
			checkRecordIsReadable(o.Ops, c.readsRecord),
			checkSearchIsCompilable(o.Ops, c.predicateErr),
		} {
			if check != nil {
				return nil, fmt.Errorf("operations[%d]: %w", i, check)
			}
		}

		// Resolved once, here, so that a decision and the log of it report the
		// mode the policy runs in rather than the mode the operator omitted.
		mode := o.Mode
		if mode == "" {
			mode = policyconfig.ModeEnforce
		}

		for _, op := range o.Ops {
			policy := operationPolicy{
				id:        o.PolicyID(),
				mode:      mode,
				validator: c.validator,
				verdicts:  c.verdicts,
			}

			// Only at search is the policy compiled rather than evaluated,
			// so only there is there a predicate to carry.
			if op == policyconfig.OpSearch {
				policy.predicate = c.predicate
			}

			byOp[op] = append(byOp[op], policy)
		}
	}

	return byOp, nil
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
// Only pull and search decide about records the catalogue already holds, so
// only they have verdicts. At push the record is arriving and no evaluation
// has seen it; at index it is being made searchable, which is what has to
// happen before an evaluation can run; at autosync it is someone else's
// record arriving. Reading verdicts at any of those would be reading an empty
// map, and the policy would refuse everything while appearing to decide.
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
		if op != policyconfig.OpPull && op != policyconfig.OpSearch {
			return fmt.Errorf("reads verdicts, which only ops %q and %q have: op %q decides before any verdict exists",
				policyconfig.OpPull, policyconfig.OpSearch, op)
		}
	}

	return nil
}

// checkRecordIsReadable rejects a policy that reads the record's content at
// an op that has no record to read.
//
// A read by CID is gated before the content is fetched, which is what keeps a
// lookup an index read. A search has no one record at all: it decides which
// records there will be. Handing either an empty record would let a policy
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

	for _, op := range []string{policyconfig.OpPull, policyconfig.OpSearch} {
		if slices.Contains(ops, op) {
			return fmt.Errorf("reads the record at op %q, which decides before the record is read: "+
				"put a rule about record content in an evaluation policy and read its verdict here", op)
		}
	}

	return nil
}

// checkSearchIsCompilable rejects a policy at search that says something a
// query cannot.
//
// Every other op evaluates its policy, so anything the language can express
// is available. A search has to decide before rows are selected, so its
// policy becomes part of the query, and only a boolean combination of verdict
// terms survives that.
func checkSearchIsCompilable(ops []string, predicateErr error) error {
	if predicateErr == nil || !slices.Contains(ops, policyconfig.OpSearch) {
		return nil
	}

	return fmt.Errorf("cannot be used at op %q: %w", policyconfig.OpSearch, predicateErr)
}

// provider is the settings of whichever backend is configured. Config
// validation has already rejected the ones that do not apply.
type provider struct {
	schemaURL   string
	file        string
	expressions []string
}

// evaluationPolicyIDs returns the evaluation policy IDs in config order.
// A verdict key is resolved against this list.
func evaluationPolicyIDs(evaluations policyconfig.Evaluations) []string {
	ids := make([]string, len(evaluations))
	for i, e := range evaluations {
		ids[i] = e.PolicyID()
	}

	return ids
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
// l is the layer the policy was configured in. A provider that compiles its
// policy rejects reading the other layer's inputs at load time.
// evaluationIDs are the configured evaluation policies a verdict key may name.
func newValidator(ctx context.Context, l layer, name string, p provider, policyDir string, evaluationIDs []string) (compiled, error) {
	switch name {
	case policyconfig.ProviderOASF:
		v, err := newOASFValidator(p.schemaURL)
		if err != nil {
			return compiled{}, fmt.Errorf("failed to initialize OASF validator: %w", err)
		}

		// OASF is nothing but a reading of the record: it checks the content
		// against a schema. There is no sense in which it reads less.
		return compiled{
			validator:    v,
			readsRecord:  true,
			predicateErr: errNotCompilableProvider(policyconfig.ProviderOASF),
		}, nil
	case policyconfig.ProviderCEL:
		v, err := newCELValidatorWithIDs(p.expressions, l, evaluationIDs)
		if err != nil {
			return compiled{}, fmt.Errorf("failed to initialize CEL validator: %w", err)
		}

		return compiled{
			validator:    v,
			version:      v.version,
			factGroups:   v.factGroups,
			verdicts:     v.verdicts,
			readsRecord:  v.readsRecord,
			predicate:    v.predicate,
			predicateErr: v.predicateErr,
		}, nil
	case policyconfig.ProviderOPA:
		v, err := newOPAValidator(ctx, policyDir, p.file, l, evaluationIDs)
		if err != nil {
			return compiled{}, fmt.Errorf("failed to initialize OPA validator: %w", err)
		}

		return compiled{
			validator:    v,
			version:      v.version,
			factGroups:   v.factGroups,
			verdicts:     v.verdicts,
			readsRecord:  v.readsRecord,
			predicateErr: errNotCompilableProvider(policyconfig.ProviderOPA),
		}, nil
	default:
		return compiled{}, fmt.Errorf("unsupported provider %q", name)
	}
}

// SearchPolicy is a policy at op search as the registry holds it: compiled,
// but not yet joined to the versions of the policies it reads.
//
// Joining it is somebody else's job because the registry cannot do it. The
// version a policy is in force at, and whether its verdicts cover the
// catalogue, are both read from the database.
type SearchPolicy struct {
	ID        string
	Mode      policyconfig.Mode
	Predicate predicate.Node

	// Reads are the evaluation policies the predicate names, sorted and
	// without repeats. The caller owns the slice.
	Reads []string
}

// SearchPolicies returns the policies configured at op search, in config
// order, skipping the ones switched off.
func (r *Registry) SearchPolicies() []SearchPolicy {
	if r == nil {
		return nil
	}

	var policies []SearchPolicy

	for _, p := range r.byOp[policyconfig.OpSearch] {
		if p.mode == policyconfig.ModeOff {
			continue
		}

		// A key that could not be read widens to every evaluation policy,
		// the same way mayEnforce widens, so the policy waits for all of
		// them rather than for none.
		//
		// No configuration reaches it today: validateAst refuses every way
		// of reaching verdicts that no literal key can be read from, and a
		// search policy is CEL by the time it gets here. It stays because
		// removing it is silent. keys is empty in exactly the case all is
		// set, so the fallback would be a policy that waits for nothing,
		// passes the backfill latch at once, and hides every record no
		// verdict covers — which is what the latch exists to prevent.
		reads := slices.Clone(p.verdicts.keys)
		if p.verdicts.all {
			reads = r.evaluationIDs()
		}

		policies = append(policies, SearchPolicy{
			ID:        p.id,
			Mode:      p.mode,
			Predicate: p.predicate,
			Reads:     reads,
		})
	}

	return policies
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

	ok, msgs, _, err := r.apply(ctx, op, input, func(ctx context.Context, v Validator) (bool, []string, error) {
		return validateOperation(ctx, record, input, v)
	})

	return ok, msgs, err
}

// DecidePull applies the operation policies at pull to a record known only by
// its CID, which is all a read by CID has: the gate runs before the content
// is fetched, so that a lookup stays an index read.
//
// A policy that would read the record is refused at load time, so what is
// missing here is only what nothing configured can ask for.
//
// The third return is apply's: the mode of the strictest refusal, which the
// gate counts the exclusion under.
func (r *Registry) DecidePull(ctx context.Context, recordCID string,
) (bool, []string, policyconfig.Mode, error) {
	if r == nil || len(r.byOp[policyconfig.OpPull]) == 0 {
		return true, nil, "", nil
	}

	var verdicts map[string]*policyv1.Verdict

	if r.readsVerdicts(policyconfig.OpPull) {
		read, err := r.readVerdicts(recordCID)
		if err != nil {
			// A check nothing would act on cannot change the answer, so a
			// failure to make it is reported and the record served. Shadow
			// is for learning what a policy would do, not for failing reads
			// while finding out.
			if !r.actsAt(policyconfig.OpPull) {
				logger.Warn("Could not read the verdicts a pull policy checks, and none is enforcing",
					"cid", recordCID, "error", err)

				return true, nil, "", nil
			}

			return false, nil, "", err
		}

		verdicts = read
	}

	input := pullInput(recordCID, verdicts)

	ok, msgs, refused, err := r.apply(ctx, policyconfig.OpPull, input,
		func(ctx context.Context, v Validator) (bool, []string, error) {
			return validateOperationByCID(ctx, input, v)
		})

	return ok, msgs, refused, err
}

// actsAt reports whether a refusal at op would change any answer: whether
// some policy there is enforcing, the backfill latch having had its say.
func (r *Registry) actsAt(op string) bool {
	for _, p := range r.byOp[op] {
		if p.validator == nil || p.mode != policyconfig.ModeEnforce {
			continue
		}

		if r.mayEnforce(p.verdicts) {
			return true
		}
	}

	return false
}

// apply runs each policy at op and collects what the enforcing ones refused.
//
// run is how one policy is evaluated, which differs by what the caller has:
// a record to measure and time out against, or only a CID.
//
// The third return is the mode of the strictest refusal: ModeEnforce when an
// enforcing policy refused, ModeShadow when only a shadow one would have, and
// empty when every policy passed. A shadow refusal changes no answer, so it
// is reported separately rather than through the first two returns, and it
// is the whole of what a shadow policy produces besides its log line.
func (r *Registry) apply(
	ctx context.Context,
	op string,
	input *policyv1.OperationInput,
	run func(context.Context, Validator) (bool, []string, error),
) (bool, []string, policyconfig.Mode, error) {
	var (
		all     []string
		refused policyconfig.Mode
	)

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
			return false, nil, "", fmt.Errorf("decide record with policy %q: %w", p.id, err)
		}

		logDecision(ctx, operationPolicy{id: p.id, mode: mode, validator: p.validator}, input, ok, msgs)

		if ok {
			continue
		}

		if mode == policyconfig.ModeEnforce {
			all = append(all, msgs...)
		}

		// An enforcing refusal outranks a shadow one: the record was
		// excluded, not merely would have been.
		if refused != policyconfig.ModeEnforce {
			refused = mode
		}
	}

	return len(all) == 0, all, refused, nil
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
