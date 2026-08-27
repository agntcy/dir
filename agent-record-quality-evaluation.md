# Agent Record Quality Evaluation: MCP, Agent Skills, and A2A

**Context:** The `dir` production node hosts hundreds of OASF records describing MCP servers, Agent Skills, and A2A agents, with no systematic way to assess whether those records are live, well-formed, or safe. This report surveys what "quality" means for each protocol, maps existing open source tooling to each quality parameter, and flags where no adequate tool exists today.

**Date:** 2026-08-27

---

## 1. What "Quality" Means, By Protocol

### 1.1 MCP (Model Context Protocol)

| Dimension | What it means |
|---|---|
| Liveness / reachability | The endpoint responds at all |
| Handshake / protocol conformance | `initialize`, capability negotiation, `tools/list`, `resources/list`, `prompts/list` match the spec |
| Tool schema validity | Tool `inputSchema` fields are well-formed JSON Schema |
| Tool description security | No prompt injection or "tool poisoning" hidden in tool metadata/descriptions |
| Rug-pull / drift detection | Tool definitions haven't silently changed since last approval/evaluation |
| Auth posture | None vs. static API key vs. OAuth 2.1; whether secrets are exposed |
| Over-privileged capability auditing | Whether a tool requests more access/capability than its stated function needs |
| Behavioral correctness | Whether the tool actually does what its description claims (functional fidelity) |
| Latency / performance | Response time, uptime history |

### 1.2 Agent Skills (SKILL.md)

| Dimension | What it means |
|---|---|
| Frontmatter / format compliance | Required fields present, name/description length limits, no disallowed keys |
| Cross-agent schema compatibility | Valid against the target agent's expected schema (Claude, and others where applicable) |
| Referenced-resource integrity | Bundled scripts/files the SKILL.md points to actually exist and are valid |
| Prompt injection / hidden instructions | The skill body doesn't smuggle instructions outside its stated purpose |
| Malicious/dangerous bundled code | No executable payloads, data exfiltration, or supply-chain risk in bundled scripts |
| Excessive permissions | The skill doesn't request broader access than its task requires |
| Publishing readiness | Marketplace/publishing metadata is complete |
| Behavior-to-description match | Whether the skill's actual instructions match what it claims to do |

### 1.3 A2A (Agent2Agent)

| Dimension | What it means |
|---|---|
| Agent Card schema compliance | Required fields present; skill definitions well-formed |
| Card reachability | The Agent Card (e.g. `/.well-known/agent.json`) actually resolves |
| Live protocol conformance | The agent's runtime implements A2A JSON-RPC/gRPC/REST methods per spec |
| Capability truthfulness | Declared capabilities (streaming, push notifications, etc.) actually work |
| Auth scheme correctness | Declared authentication is actually enforced as described |
| Security (card/skill poisoning) | No injected instructions in skill descriptions/examples; no impersonation risk |
| Interoperability / functional exchange | An end-to-end multi-turn task actually completes |

---

## 2. Open Source Tooling, By Quality Parameter

Maintenance status below is based on live GitHub data pulled on 2026-08-27 (stars, last push, archived status). "Official" means the tool is maintained by the organization that owns the protocol spec itself.

### 2.1 MCP

| Quality parameter | Tool | Official? | Maintenance |
|---|---|---|---|
| Handshake / protocol conformance (interactive debugging) | [modelcontextprotocol/inspector](https://github.com/modelcontextprotocol/inspector) | **Yes** — maintained by the MCP spec org | Excellent — 10.8k★, pushed today |
| Handshake / protocol conformance (automated) | [Janix-ai/mcp-validator](https://github.com/Janix-ai/mcp-validator) | No | Slowing — 78★, last push ~6 months ago; verify against current spec version before relying on it |
| Tool description security / tool poisoning / rug-pull detection | [cisco-ai-defense/mcp-scanner](https://github.com/cisco-ai-defense/mcp-scanner) | No (vendor) | Healthy — 1,051★, pushed yesterday |
| Tool description security / tool poisoning / rug-pull detection | [snyk/agent-scan](https://github.com/snyk/agent-scan) (formerly Invariant Labs' `mcp-scan` — same repo, transferred after Snyk's acquisition) | No (vendor) | Healthy — 2,964★, pushed today |
| Liveness / reachability | [openstatus](https://github.com/openstatusHQ/openstatus) | No | Very healthy — 9,024★, pushed today; general uptime platform with an MCP-aware JSON-RPC health check, not MCP-native |
| Behavioral correctness (functional/LLM-judge testing) | [lastmile-ai/mcp-eval](https://github.com/lastmile-ai/mcp-eval) | No | Going stale — 35★, last push ~9 months ago |
| Auth posture | Partial coverage inside `mcp-scanner` / `agent-scan` | No | See above |

### 2.2 Agent Skills

| Quality parameter | Tool | Official? | Maintenance |
|---|---|---|---|
| Frontmatter / format compliance | [himself65/skill-lint](https://github.com/himself65/skill-lint) | No | Small but active — 14★, pushed 12 days ago |
| Frontmatter / format compliance | [William-Yeh/agent-skill-linter](https://github.com/William-Yeh/agent-skill-linter) | No | Weak — 10★, last push ~3 months ago |
| Prompt injection / hidden instructions | [cisco-ai-defense/skill-scanner](https://github.com/cisco-ai-defense/skill-scanner) | No (vendor) | Healthy — 2,461★, pushed 3 weeks ago |
| Prompt injection / hidden instructions | [snyk/agent-scan](https://github.com/snyk/agent-scan) (Skill Inspector) | No (vendor) | Healthy — 2,964★, pushed today |
| Prompt injection / hidden instructions | [getsentry/skills](https://github.com/getsentry/skills) (`skill-scanner`) | No (vendor) | Healthy — 956★, pushed 2 days ago |
| Malicious bundled code / excessive permissions | `cisco-ai-defense/skill-scanner`, `snyk/agent-scan` | No | Same as above |
| Spec reference / template | [anthropics/skills](https://github.com/anthropics/skills) | **Yes** — the spec-owning org | Very active, but ships **no validator or linter** — spec + template only |

### 2.3 A2A

| Quality parameter | Tool | Official? | Maintenance |
|---|---|---|---|
| Agent Card schema compliance (static) | [a2aproject/a2a-inspector](https://github.com/a2aproject/a2a-inspector) | **Yes** — Linux Foundation A2A project | Active — 479★, pushed 3 weeks ago |
| Live protocol conformance (MUST/SHOULD/MAY, 3 transports) | [a2aproject/a2a-tck](https://github.com/a2aproject/a2a-tck) | **Yes** — official Technology Compatibility Kit | Moderate — 48★, pushed ~2 months ago (low star count is normal for a TCK) |
| Capability truthfulness | `a2a-tck` "Capabilities" test category | **Yes** | Same as above |
| Interoperability / functional exchange | `a2a-inspector` (interactive), `a2a-tck` (automated) | **Yes** | See above |
| Security (card/skill poisoning) | [cisco-ai-defense/a2a-scanner](https://github.com/cisco-ai-defense/a2a-scanner) | No (vendor) | Reasonable — 164★, last push ~4 months ago; newest and least battle-tested of Cisco's three scanners |
| Agent Card schema compliance (CI-friendly, alternative) | capiscio/validate-a2a (GitHub Action) | No | **Not recommended** — 1★, minimal adoption, not GitHub-certified |

---

## 3. Quality Parameters With No Adequate Tool

| Protocol | Parameter | Status |
|---|---|---|
| MCP | Over-privileged capability auditing (a tool requesting more access than it needs) | Only academic coverage — [arXiv: "Auditing MCP Servers for Over-Privileged Tool Capabilities"](https://arxiv.org/pdf/2603.21641) — no shipped OSS tool |
| MCP | Auth posture as a first-class, standalone check (vs. a side-effect of a broader security scan) | No dedicated tool; only partial/manual coverage. A 2026 BlueRock Security study of ~7,000 public MCP servers found 36.7% SSRF-vulnerable, 41% requiring no authentication, and only 8.5% using OAuth — a real, documented problem with no purpose-built OSS checker |
| Skills | Referenced-resource integrity (do bundled scripts/files a SKILL.md points to actually exist and run) | No linter surveyed explicitly checks this — worth verifying directly against the record set |
| Skills | Prompt injection / malicious-instruction detection, at production-grade accuracy | Tools exist (Cisco, Snyk, Sentry) but the field is unsettled: one benchmark found existing scanners averaging only ~55–65% accuracy (ClawGuard Auditor 65%, SlowMist 64.2%, Cisco Skill Scanner 63.8%, Skill Vetter 53.1%), while other academic tools (SkillSieve, Skill-guard) report much higher F1 on their own benchmarks. Numbers are inconsistent across papers because each self-reports against a different labeled set — no tool here should be treated as authoritative |
| A2A | Behavior-to-description match (does the agent actually do what its Card/skills claim) | No tool found — `a2a-tck` checks protocol conformance, not semantic truthfulness of skill descriptions |

---

## 4. Recommendation

The reconciler has a mature, actively-developed `scan` task (`reconciler/tasks/scan/`, backed by a shared `utils/scanner/` package also used by the CLI importer) that:

- runs three external scanner CLIs — `mcp-scanner`, `skill-scanner`, `a2a-scanner` — per record, on a 6-hour interval with a 7-day per-record TTL, opt-in via config (`Enabled` defaults to `false`)
- for **MCP**: scans both the record's declared source repo (LLM-backed behavioral analysis) *and* dials live `sse`/`streamable-http` endpoints directly, with SSRF-hardened URL validation (blocks private/loopback/cloud-metadata ranges) and a per-record cap on endpoints scanned
- for **A2A**: scans only the static AgentCard JSON embedded in the record — a live-endpoint A2A scan is explicitly flagged as unbuilt follow-up work in the code (`utils/scanner/a2a.go`)
- for **Skills**: scans the extracted skill-bundle files locally; no network call involved
- persists results both as signed OCI referrers on the record (`agntcy.dir.security.v1.ScanReport`) and as summary rows in a `scan_reports` DB table, with `is_safe`/`max_severity` exposed as Search predicates

This already matches the vendor lineage this report recommends — the tool names correspond to Cisco AI Defense's `mcp-scanner`/`skill-scanner`/`a2a-scanner` family — and already implements the OASF-locator dispatch + normalized per-record scoring this report was going to propose building from scratch.

**Given that, the real next steps are narrower — closing the specific gaps identified above within the existing task, not standing up new infrastructure:**

1. **Close the A2A live-endpoint gap.** This is the one concrete, code-confirmed hole: A2A records get static Agent Card validation but no live protocol check. Extending the existing A2A runner with the official `a2aproject/a2a-tck` (or `a2a-inspector`) would mirror how the MCP runner already dials live endpoints, and directly closes a gap this report independently identified from the tooling landscape alone.
2. **Add MCP protocol-conformance, not just security.** The current MCP runner checks for malicious behavior; it doesn't check spec conformance (handshake correctness, schema validity). `modelcontextprotocol/inspector` (official) or `mcp-validator` could be layered in as a separate, lighter check alongside the existing security scan.
3. **Add Skill format/integrity linting alongside the existing security scan.** `skill-scanner` looks for malicious content, not frontmatter compliance or whether resources referenced in a SKILL.md actually exist. `skill-lint` is a cheap complementary check, not a replacement.
4. **Drift/rug-pull detection is nearly free to add.** Since `scan_reports` already stores a timestamped report per record per run, diffing consecutive reports for the same record is a small addition on top of existing data, not new infrastructure.
5. **Over-privileged capability auditing (MCP) and a dedicated auth-posture check remain open gaps** with no available OSS tool (Section 3) — these would need custom logic regardless of anything above, if they're prioritized.
