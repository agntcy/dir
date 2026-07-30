---
hide:

  - navigation
  - toc

---

<div class="dirctl-terminal-intro-group" id="dirctl-terminal-intros" markdown="1">

Publish and discover agent records in a few commands. Install the real CLI from the [Quickstart](dir/dir-quickstart.md).
{: .dirctl-terminal-intro data-intro-level="cli" hidden}

Use `dirctl --help` to see available commands. See the [CLI Reference](dir/dir-cli-reference.md) for more details.
{: .dirctl-terminal-intro data-intro-level="try" hidden}

Need a skill, MCP server, or A2A partner? Your agent searches the Directory, wires it in, and uses it right away. See the [MCP server](dir/dir-component-mcp-server.md) for more details.
{: .dirctl-terminal-intro data-intro-level="agent"}

</div>

<div class="dir-landing">

<section class="dir-hero">
  <div class="dir-hero__inner">
    <h1 class="dir-hero__title">Agent Directory Service</h1>
    <div class="dir-hero__partner">
      <span class="dir-hero__partner-text">part of</span>
      <a
        href="https://www.linuxfoundation.org/press/linux-foundation-welcomes-the-agntcy-project-to-standardize-open-multi-agent-system-infrastructure-and-break-down-ai-agent-silos"
        target="_blank"
        rel="noopener noreferrer"
      >
        <picture>
          <source
            media="(max-width: 59.9375em)"
            srcset="assets/lf-stacked-white.png"
          />
          <img
            src="assets/lf-horizontal-white.png"
            alt="Linux Foundation"
            class="dir-hero__partner-logo"
          />
        </picture>
      </a>
    </div>
    <p class="dir-hero__tagline">
      One federated registry for <br class="dir-hero__tagline-break" aria-hidden="true" />cross-<span class="dir-hero__flip-wrap"><span class="dir-hero__flip" data-words="framework,protocol,registry" aria-live="polite">framework</span></span> agent discovery.
    </p>
    <p class="dir-hero__lede">
      An open-source, framework-agnostic registry for agentic resource discovery and management.
      Publish, verify, and discover MCP servers, A2A agents, and Agent Skills through a federated control plane, enabling
      native interoperability for complex, multi-agent workflows. Agent Directory implements the
      <a href="https://agenticresourcediscovery.org/" target="_blank" rel="noopener noreferrer">ARD specification</a>.
    </p>
    <div class="dir-hero__actions">
      <div class="dir-hero__actions-main">
        <a class="dir-hero__btn" href="#quick-start">
          Quickstart
          <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M12 4l-1.41 1.41L16.17 11H4v2h12.17l-5.58 5.59L12 20l8-8z"/></svg>
        </a>
        <a class="dir-hero__btn" href="#community">
          Community
          <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M12 4l-1.41 1.41L16.17 11H4v2h12.17l-5.58 5.59L12 20l8-8z"/></svg>
        </a>
        <a class="dir-hero__btn" href="https://github.com/agntcy/dir" target="_blank" rel="noopener noreferrer">
          GitHub
          <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M12 .5C5.73.5.5 5.73.5 12c0 5.08 3.29 9.39 7.86 10.91.58.11.79-.25.79-.56 0-.28-.01-1.02-.02-2-3.2.7-3.88-1.54-3.88-1.54-.53-1.34-1.29-1.7-1.29-1.7-1.05-.72.08-.71.08-.71 1.16.08 1.77 1.19 1.77 1.19 1.03 1.77 2.7 1.26 3.36.96.1-.75.4-1.26.73-1.55-2.55-.29-5.23-1.28-5.23-5.69 0-1.26.45-2.29 1.19-3.1-.12-.29-.52-1.46.11-3.05 0 0 .97-.31 3.18 1.18a11.1 11.1 0 0 1 5.8 0c2.2-1.49 3.17-1.18 3.17-1.18.63 1.59.23 2.76.11 3.05.74.81 1.19 1.84 1.19 3.1 0 4.42-2.69 5.39-5.25 5.68.41.36.78 1.06.78 2.14 0 1.55-.01 2.8-.01 3.18 0 .31.21.68.8.56A11.51 11.51 0 0 0 23.5 12C23.5 5.73 18.27.5 12 .5z"/></svg>
        </a>
      </div>
      <a class="dir-hero__btn" href="https://ai-catalog.outshift.io/" target="_blank" rel="noopener noreferrer">
        Try Cisco's AI Catalog
        <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M19 19H5V5h7V3H5a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14c1.1 0 2-.9 2-2v-7h-2v7zM14 3v2h3.59l-9.83 9.83 1.41 1.41L19 6.41V10h2V3h-7z"/></svg>
      </a>
    </div>
  </div>
</section>

<div class="dir-features">
  <div class="dir-feature-card">
    <div class="dir-feature-card__art">
      <img src="assets/landing/feature-discovery.svg" alt="" loading="lazy" />
    </div>
    <p class="dir-feature-card__title">Capability-Based Discovery</p>
    <p class="dir-feature-card__text">
      Publish and find agents by structured skills and attributes using OASF taxonomies
      and content routing across a distributed network of directory servers.
    </p>
  </div>
  <div class="dir-feature-card">
    <div class="dir-feature-card__art">
      <img src="assets/landing/feature-federation.svg" alt="" loading="lazy" />
    </div>
    <p class="dir-feature-card__title">Federated Architecture</p>
    <p class="dir-feature-card__text">
      Interconnect directory instances through DHT-based content routing, enabling
      decentralized discovery without a single central registry.
    </p>
  </div>
  <div class="dir-feature-card">
    <div class="dir-feature-card__art">
      <img src="assets/landing/feature-trust.svg" alt="" loading="lazy" />
    </div>
    <p class="dir-feature-card__title">Verifiable Claims</p>
    <p class="dir-feature-card__text">
      Cryptographic integrity and provenance for directory records help users make
      informed decisions about agent selection and trust.
    </p>
  </div>
</div>

<section class="dir-howto">
  <div
    class="dir-graph-wrap"
    data-dir-graph
    role="img"
    aria-label="Agent Directory Service workflow: publish, discover, and verify"
  ></div>
</section>

<section class="dir-quickstart">
<h2 class="dir-section-title" id="quick-start">Quickstart</h2>

<section class="dirctl-terminal-section">
  <div class="dirctl-terminal-layout">
    <div class="dirctl-terminal-main">
      <div class="dirctl-terminal" data-mode="demo">
        <div class="dirctl-terminal-bar">
          <span class="dirctl-terminal-title">agent@workspace:~</span>
          <div class="dirctl-terminal-controls" aria-hidden="true">
            <span class="dirctl-terminal-control">&#8211;</span>
            <span class="dirctl-terminal-control dirctl-terminal-control-close">&#10005;</span>
          </div>
        </div>
        <pre
          class="dirctl-terminal-output"
          id="dirctl-terminal-output"
          aria-live="polite"
          aria-label="Terminal output"
        ></pre>
        <form class="dirctl-terminal-input" hidden>
          <label
            class="dirctl-terminal-prompt"
            for="dirctl-terminal-command"
          >user@dir:~$</label>
          <input
            id="dirctl-terminal-command"
            class="dirctl-terminal-command"
            type="text"
            autocomplete="off"
            spellcheck="false"
            aria-label="Enter a dirctl command"
          />
        </form>
      </div>
    </div>
    <div class="dirctl-terminal-side">
      <div class="dirctl-terminal-actions">
        <button type="button" class="dirctl-terminal-btn is-active" data-demo-level="agent">With your agent</button>
        <button type="button" class="dirctl-terminal-btn" data-demo-level="cli">CLI basics</button>
        <button type="button" class="dirctl-terminal-btn" data-mode-switch="try">Try it yourself</button>
        <button type="button" class="dirctl-terminal-btn dirctl-terminal-reopen" hidden>Reopen terminal</button>
      </div>
    </div>
  </div>
</section>
</section>

<section class="dir-community" id="community">
  <h2 class="dir-section-title">Community</h2>
  <p class="dir-community__lede">
    Connect with AGNTCY contributors, join working group meetings, and help shape open agent discovery.
    For more information, see the <a href="community.md">community page</a>.
  </p>

  <div class="dir-community-social">
    <a
      class="dir-community-card"
      href="https://discord.gg/FbEnSHXD34"
      target="_blank"
      rel="noopener noreferrer"
    >
      <span class="dir-community-card__icon" aria-hidden="true">
        <svg viewBox="0 0 24 24" role="img"><path d="M20.317 4.37a19.791 19.791 0 0 0-4.885-1.515.074.074 0 0 0-.079.037 12.3 12.3 0 0 0-.608 1.25 18.27 18.27 0 0 0-5.487 0 12.64 12.64 0 0 0-.617-1.25.077.077 0 0 0-.079-.037A19.736 19.736 0 0 0 3.677 4.37a.07.07 0 0 0-.032.027C.533 9.046-.32 13.58.099 18.057a.082.082 0 0 0 .031.057 19.9 19.9 0 0 0 5.993 3.03.078.078 0 0 0 .084-.028 14.09 14.09 0 0 0 1.226-1.994.076.076 0 0 0-.041-.106 13.107 13.107 0 0 1-1.872-.892.077.077 0 0 1-.008-.128 10.2 10.2 0 0 0 .372-.292.074.074 0 0 1 .077-.01c3.928 1.793 8.18 1.793 12.062 0a.074.074 0 0 1 .078.01c.12.098.246.198.373.292a.077.077 0 0 1-.006.127 12.299 12.299 0 0 1-1.873.892.077.077 0 0 0-.041.107c.36.698.772 1.362 1.225 1.993a.076.076 0 0 0 .084.028 19.839 19.839 0 0 0 6.002-3.03.077.077 0 0 0 .032-.054c.5-5.177-.838-9.674-3.549-13.66a.061.061 0 0 0-.031-.03zM8.02 15.33c-1.183 0-2.157-1.085-2.157-2.419 0-1.333.956-2.419 2.157-2.419 1.21 0 2.176 1.096 2.157 2.42 0 1.333-.956 2.418-2.157 2.418zm7.975 0c-1.183 0-2.157-1.085-2.157-2.419 0-1.333.955-2.419 2.157-2.419 1.21 0 2.176 1.096 2.157 2.42 0 1.333-.946 2.418-2.157 2.418z"/></svg>
      </span>
      <span class="dir-community-card__body">
        <span class="dir-community-card__title">Discord</span>
        <span class="dir-community-card__text">Chat with maintainers and contributors in the Agent Directory Discord server.</span>
      </span>
    </a>

    <a
      class="dir-community-card"
      href="https://zoom-lfx.platform.linuxfoundation.org/meetings/agntcy?view=week"
      target="_blank"
      rel="noopener noreferrer"
    >
      <span class="dir-community-card__icon" aria-hidden="true">
        <svg viewBox="0 0 24 24" role="img"><path d="M19 4h-1V2h-2v2H8V2H6v2H5a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2V6a2 2 0 0 0-2-2zm0 16H5V10h14v10zM5 8V6h14v2H5zm2 4h10v2H7v-2zm0 4h7v2H7v-2z"/></svg>
      </span>
      <span class="dir-community-card__body">
        <span class="dir-community-card__title">Meetings</span>
        <span class="dir-community-card__text">Join working group and community meetings on the AGNTCY calendar.</span>
      </span>
    </a>

    <a
      class="dir-community-card"
      href="https://blogs.agntcy.org/"
      target="_blank"
      rel="noopener noreferrer"
    >
      <span class="dir-community-card__icon" aria-hidden="true">
        <svg viewBox="0 0 24 24" role="img"><path d="M19 3H5a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2V5a2 2 0 0 0-2-2zm-5 14H7v-2h7v2zm3-4H7v-2h10v2zm0-4H7V7h10v2z"/></svg>
      </span>
      <span class="dir-community-card__body">
        <span class="dir-community-card__title">Blog</span>
        <span class="dir-community-card__text">Read announcements, tutorials, and technical deep dives from the Agent Directory team.</span>
      </span>
    </a>
  </div>

  <div class="dir-community-contribute">
    <div class="dir-community-contribute__text">
      <h3 class="dir-community-contribute__title">Contribute</h3>
      <p>
        Help build the Agent Directory Service by contributing code, reporting bugs, or suggesting
        enhancements. Pick up a good first issue, review open pull requests, or read the
        contributing guide to get started.
      </p>
      <div class="dir-community-contribute__actions">
        <a
          class="dir-community-contribute__btn"
          href="https://github.com/agntcy/dir"
          target="_blank"
          rel="noopener noreferrer"
        >
          Visit our GitHub
          <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M12 .5C5.73.5.5 5.73.5 12c0 5.08 3.29 9.39 7.86 10.91.58.11.79-.25.79-.56 0-.28-.01-1.02-.02-2-3.2.7-3.88-1.54-3.88-1.54-.53-1.34-1.29-1.7-1.29-1.7-1.05-.72.08-.71.08-.71 1.16.08 1.77 1.19 1.77 1.19 1.03 1.77 2.7 1.26 3.36.96.1-.75.4-1.26.73-1.55-2.55-.29-5.23-1.28-5.23-5.69 0-1.26.45-2.29 1.19-3.1-.12-.29-.52-1.46.11-3.05 0 0 .97-.31 3.18 1.18a11.1 11.1 0 0 1 5.8 0c2.2-1.49 3.17-1.18 3.17-1.18.63 1.59.23 2.76.11 3.05.74.81 1.19 1.84 1.19 3.1 0 4.42-2.69 5.39-5.25 5.68.41.36.78 1.06.78 2.14 0 1.55-.01 2.8-.01 3.18 0 .31.21.68.8.56A11.51 11.51 0 0 0 23.5 12C23.5 5.73 18.27.5 12 .5z"/></svg>
        </a>
        <a
          class="dir-community-contribute__btn dir-community-contribute__btn--highlight"
          href="https://github.com/search?q=org%3Aagntcy+type%3Aissue+label%3A%22good-first-issue%22%2C%22good+first+issue%22&type=issues"
          target="_blank"
          rel="noopener noreferrer"
        >
          Good first issues
        </a>
      </div>
    </div>
    <div class="dir-community-contribute__metrics">
      <a
        class="dir-repobeats__link"
        href="https://github.com/agntcy/dir/pulse?period=monthly"
        target="_blank"
        rel="noopener noreferrer"
        data-dir-repobeats
        data-repo="agntcy/dir"
      >
        <div class="dir-repobeats" aria-busy="true" aria-label="GitHub repository metrics">
          <p class="dir-repobeats__loading">Loading repository metrics…</p>
        </div>
      </a>
    </div>
  </div>
</section>

</div>

<section class="dir-resources" markdown="1">

<h2 class="dir-section-title">Explore Agent Directory Service</h2>

<div class="grid cards" markdown>

- :material-rocket-launch:{ .lg .middle } **Quickstart**

    Run a local Directory instance in minutes.

    [:octicons-arrow-right-24: Quickstart](dir/dir-quickstart.md)

- :material-book-open:{ .lg .middle } **Read the Introduction**

    Understand core concepts, architecture, and features.

    [:octicons-arrow-right-24: Overview](dir/dir-overview.md)

    [:octicons-arrow-right-24: Architecture](dir/dir-architecture.md)

- :material-file-document-outline:{ .lg .middle } **Dive into the Specification**

    Explore the ADS Internet Draft and protocol definition.

    [:octicons-arrow-right-24: ADS Specification](https://datatracker.ietf.org/doc/draft-mp-agntcy-ads)

    [:octicons-arrow-right-24: AI Catalog specification](https://agent-card.github.io/ai-catalog/)

- :material-code-braces:{ .lg .middle } **SDKs and Tools**

    Client libraries, CLI, and API references.

    [:octicons-arrow-right-24: SDK Overview](dir/dir-sdk.md)

    [:octicons-arrow-right-24: CLI Reference](dir/dir-cli-reference.md)

- :material-cloud-upload:{ .lg .middle } **Deploy**

    Local, Kubernetes, and production deployment guides.

    [:octicons-arrow-right-24: Local Deployment](dir/dir-deployment-local.md)

    [:octicons-arrow-right-24: Production Deployment](dir/dir-prod-deployment.md)

- :material-newspaper-variant-outline:{ .lg .middle } **Linux Foundation Press Release**

    Read how the Linux Foundation welcomed the AGNTCY project to standardize open
    multi-agent system infrastructure and break down AI agent silos.

    [:octicons-arrow-right-24: LF press release](https://www.linuxfoundation.org/press/linux-foundation-welcomes-the-agntcy-project-to-standardize-open-multi-agent-system-infrastructure-and-break-down-ai-agent-silos)

- :material-lan-connect:{ .lg .middle } **Join the Federation Testbed**

    We invite organizations, researchers, and developers to join the Agent
    Directory Testbed—a decentralized, open staging environment for next-generation
    AI agent discovery and secure registry federation.

    [:octicons-arrow-right-24: Call for federation partners](https://github.com/agntcy/dir/discussions/455)

    [:octicons-arrow-right-24: Federated Directory setup](dir/dir-federation-setup.md)

</div>

</section>
