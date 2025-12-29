# Trust Ranking Extension (Reference PoC)

> **Status:** Reference-only implementation  
> **Scope:** Demonstration and discussion  
> **Non-goals:** Security guarantees, standards, production readiness

This document describes a **reference trust-ranking extension** that can be used
*alongside* the AGNTCY directory. It is intentionally optional, additive, and
non-authoritative.

## Overview

The AGNTCY directory enables capability-based discovery:

> “Find agents that can do X.”

As ecosystems grow, this becomes insufficient on its own.
When many agents claim the same capability, consumers need additional signals
to decide *which* agent to try first.

This reference extension demonstrates how **trust-related signals** could
influence ranking decisions without changing directory semantics or protocol
behavior.

## Background

Identity verification answers “who is this agent?”  
Trust-related signals answer “how reliable does this agent appear to be?”

Both dimensions matter.  
This reference focuses on ranking based on the latter.

## What this extension is

- A **sidecar** ranking module
- A **toy scoring model** using simple heuristics
- A **runnable demo** that produces explainable results
- A way to explore *interfaces*, not prescribe policy

## What this extension is not

- Not a security system
- Not a standard
- Not a recommendation for production use
- Not a source of truth for trust decisions

All trust logic here is local, subjective, and replaceable.

## Architecture

```
User / Client
      |
      v
Directory Search → [Capable Agents]
                          |
                          v
                   Trust Ranking (optional)
                          |
                          v
                    [Ranked Results]
```

- The AGNTCY directory remains unchanged.
- Ranking occurs **after** discovery.
- Consumers opt in by choosing to apply a ranker.

## Scoring Model (Reference Only)

The reference implementation evaluates four dimensions.
Weights are fixed and chosen for clarity, not optimality.

### 1. Completeness (35%)

**What:** Is the agent profile reasonably complete?

**Signals:**

- presence of `id`, `name`, `url`
- non-empty `capabilities`
- contact information
- `updated_at` timestamp

**Rationale:** Complete profiles are easier to understand and maintain.

### 2. Verification (25%)

**What:** Are basic identity signals present?

**Signals:**

- `domain_verified`
- `key_present`

**Rationale:** Verification raises the cost of impersonation, even if imperfect.

### 3. Freshness (20%)

**What:** Is the profile actively maintained?

**Signals:**

- recently updated
- moderately recent
- stale

**Rationale:** Abandoned profiles correlate with broken integrations.

### 4. Behavior (20%)

**What:** Are there basic indicators of operational reliability?

**Signals (simulated in demo):**

- handshake failure ratio
- rate limit violations
- complaint flags

**Rationale:** Past behavior is often predictive of future reliability.

## Final Score

Scores are combined into a 0–100 range and capped below 100
to avoid implying certainty.

**Trust bands:**

- **Green:** high confidence
- **Yellow:** medium confidence
- **Red:** low confidence

Scores are accompanied by **human-readable reasons** to make
ranking decisions inspectable.

## Usage

### Run the demo

```bash
python scripts/run_trust_ranking.py --top 10
```

### JSON Output

```bash
python scripts/run_trust_ranking.py --json > ranked.json
```

Each returned agent may include:

```json
"trust": {
  "score": 77.0,
  "band": "yellow",
  "reasons": [
    "Profile is somewhat complete",
    "Updated this quarter",
    "No rate limit violations"
  ]
}
```

### Example Output (Illustrative)

```
1. Theta Support
   id: agent_theta_clean
   url: https://theta.example
   trust: 99.0 (green)
   reason: Profile is complete; Recently updated; Low handshake failure rate

2. Alpha Services
   id: agent_alpha_clean
   url: https://alpha.example
   trust: 99.0 (green)
   reason: Profile is complete; Recently updated; Domain verified
```

Output is dependent on local scoring logic and input data; results are not authoritative.

## Integration Patterns

### Pattern 1: Client-side ranking (recommended)

```python
# Fetch capable agents from a directory
agents = directory_search(...)

# Apply optional trust ranking locally
ranked_agents = rank_agents(agents)

# Select a preferred agent
selected = ranked_agents[0]
```

- No server or protocol changes required
- Multiple ranking models can coexist
- Trust preferences remain local to the client



### Pattern 2: Proxy service

A proxy queries the directory, applies ranking, and returns ordered results.  
Useful for shared logic, but introduces centralization tradeoffs.

### Pattern 3: Directory plugin (future)

Trust ranking as an optional server-side hook.  
This requires community discussion and governance alignment.

## Limitations

This PoC intentionally omits:

- adversarial robustness and Sybil resistance
- cryptographic binding of behavior to identity
- adaptive or context-dependent weighting
- trust decay, recovery, or volatility
- cross-observer reputation aggregation
- production concerns (scale, abuse, monitoring)

These omissions are deliberate.

## Purpose

This reference exists to support discussion around:

- where trust-based ranking should live
- how ranking logic can remain optional
- how explanations improve transparency
- how ecosystems avoid a single “trust authority”

Feedback and alternative approaches are encouraged.

## License

Apache 2.0 (same as AGNTCY dir)
