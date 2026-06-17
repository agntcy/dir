---
hide:

  - navigation
  - toc

---

# DIR Community

Contributors to the **Agent Directory Service (ADS)** work in the open on the
specification, server implementation, client SDKs, and federation deployments.
If you run a directory node, publish OASF agent records, or integrate discovery
into your stack, the links below are where the work happens—repos, Slack,
meetings, and recent posts from the team.

ADS is developed within
[AGNTCY](https://github.com/agntcy), alongside OASF, SLIM, and other
infrastructure for open, vendor-neutral agent discovery.

---

## Connect with us

Stay involved with the AGNTCY community and DIR contributors:

- **[AGNTCY Slack workspace](https://join.slack.com/t/agntcy/shared_invite/zt-3xozr6nzq-i6LXv2P8l2kVW4_Prnny2w)** — chat with maintainers and contributors
- **[AGNTCY Meeting Calendar](https://zoom-lfx.platform.linuxfoundation.org/meetings/agntcy?view=week)** — working group and community meetings
- **[AGNTCY on GitHub](https://github.com/agntcy)** — organization home for all IoA projects
- **[AGNTCY Blogs](https://blogs.agntcy.org/)** — announcements, tutorials, and project updates

---

## Recent News & Blog Posts

Stay up to date with announcements, tutorials, and technical deep dives from the DIR team and community:

- **[ARD over Agent Directory: Interoperability by Design](https://blogs.agntcy.org/technical/2026/06/17/ai-catalog-over-directory.html)** — *June 17, 2026*
- **[lazydir: A Terminal UI for Browsing Agent Directory](https://blogs.agntcy.org/technical/2026/05/20/lazydir-v0.0.1.html)** — *May 20, 2026*
- **[Directory Federation Hands-On: SPIRE and SPIFFE in a Local Kind Environment](https://blogs.agntcy.org/technical/security/directory/2026/02/25/directory-federation.html)** — *February 25, 2026*
- **[Directory MCP Server: Bringing AI Agent Discovery to Your IDE](https://blogs.agntcy.org/technical/2026/02/19/directory-mcp-server.html)** — *February 19, 2026*
- **[Agent Directory v1.0: Distributed Announce and Discovery of Multi-Agentic-Systems](https://blogs.agntcy.org/technical/2026/02/19/dir-v1.html)** — *February 19, 2026*
- **[Directory v1.0: Dual-Mode Authentication for Secure Agent Discovery](https://blogs.agntcy.org/security/authentication/directory/2026/01/22/directory-authentication-dual-mode-security.html)** — *January 22, 2026*

---

## Repositories

<div class="grid cards" markdown>

- :fontawesome-brands-github:{ .lg .middle } **[agntcy/dir](https://github.com/agntcy/dir)**

    Reference implementation — Go server and client nodes with gRPC and CLI tools.

- :fontawesome-brands-github:{ .lg .middle } **[agntcy/dir-spec](https://github.com/agntcy/dir-spec)**

    ADS specification — Internet Draft sources and protocol definition.

- :fontawesome-brands-github:{ .lg .middle } **[agntcy/oasf](https://github.com/agntcy/oasf)**

    Open Agentic Schema Framework — schema and taxonomy for agent metadata.

- :fontawesome-brands-github:{ .lg .middle } **[agntcy/oasf-sdk](https://github.com/agntcy/oasf-sdk)**

    OASF SDK — libraries for validating and working with OASF records.

- :fontawesome-brands-github:{ .lg .middle } **[agntcy/dir-importer](https://github.com/agntcy/dir-importer)**

    Import workflow — discover and publish agent records into Directory.

- :fontawesome-brands-github:{ .lg .middle } **[agntcy/dir-runtime](https://github.com/agntcy/dir-runtime)**

    Runtime discovery — watch container runtimes and expose agent metadata.

- :fontawesome-brands-github:{ .lg .middle } **[agntcy/dir-mcp](https://github.com/agntcy/dir-mcp)**

    MCP server — Model Context Protocol integration for Directory.

- :fontawesome-brands-github:{ .lg .middle } **[agntcy/dir-staging](https://github.com/agntcy/dir-staging)**

    Deployment examples — GitOps and staging configurations.

- :fontawesome-brands-github:{ .lg .middle } **[agntcy/dir-sdk-javascript](https://github.com/agntcy/dir-sdk-javascript)**

    JavaScript/TypeScript SDK for Directory clients.

- :fontawesome-brands-github:{ .lg .middle } **[agntcy/dir-sdk-python](https://github.com/agntcy/dir-sdk-python)**

    Python SDK for Directory clients.

- :fontawesome-brands-github:{ .lg .middle } **[agntcy/dir-gui](https://github.com/agntcy/dir-gui)**

    Desktop GUI for browsing and managing directory records.

- :fontawesome-brands-github:{ .lg .middle } **[agntcy/oidc-gateway](https://github.com/agntcy/oidc-gateway)**

    Policy-based OIDC gateway for Envoy — authentication for Directory deployments.

</div>

---

## Contributing

We welcome contributions across the DIR ecosystem and the broader AGNTCY project:

- Pick up a ["good first issue"](https://github.com/search?q=org%3Aagntcy+type%3Aissue+label%3A%22good-first-issue%22%2C%22good+first+issue%22&type=issues)
  across AGNTCY repositories
- Report bugs or suggest enhancements on [agntcy/dir Issues](https://github.com/agntcy/dir/issues)
- Review open pull requests on [agntcy/dir](https://github.com/agntcy/dir/pulls)

Read the AGNTCY [Contributing Guide](https://github.com/agntcy/governance/blob/main/CONTRIBUTING.md),
[Governance](https://github.com/agntcy/governance) repository, and
[Code of Conduct](https://github.com/agntcy/governance/blob/main/CODE_OF_CONDUCT.md).

---

## Related AGNTCY components

ADS works alongside other AGNTCY building blocks:

- **[OASF](https://docs.agntcy.org/oasf/open-agentic-schema-framework/)** — Open Agentic Schema Framework for agent metadata
- **[SLIM](https://docs.agntcy.org/messaging/slim-howto)** — Secure Low-Latency Interactive Messaging
