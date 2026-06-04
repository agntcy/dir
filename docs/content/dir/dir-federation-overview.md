# Federation

**Federation** connects separate Directory deployments so agents published in one instance
can be discovered and synchronized across organizational or regional boundaries. Federated
instances exchange routing metadata and records according to configured profiles and trust
policies.

## When to federate

- **Join the public network** — connect to the production Directory at
  `ads.outshift.io` to discover agents published by other participants.
- **Run a private federated instance** — deploy your own Directory and peer with selected
  partners while keeping operational control.
- **Multi-cloud or multi-region** — link EKS (or other) clusters so teams share a consistent
  agent catalog.

## Deployment guides

Federation is configured at deploy time. Choose the guide that matches your environment:

| Guide | Use case |
|-------|----------|
| [Federation on Amazon EKS](dir-federation-aws-eks.md) | End-to-end AWS deployment into the public network |
| [Running a Federated Directory Instance](dir-federation-setup.md) | Connect an existing instance to production federation |
| [Profiles](dir-federation-profiles.md) | Federation profile configuration |
| [Best Practices and Troubleshooting](dir-federation-troubleshooting.md) | Operations and debugging |

Also see [Production Deployment](dir-prod-deployment.md) and
[OIDC Authentication](dir-component-oidc-authentication.md) for gateway and external access when
federating user-facing endpoints.

## Concepts

- [Trust Model](dir-component-trust-model.md) — signing, verification, and SPIFFE/SPIRE identity
- [Routing](dir-component-routing.md) — how skill-based discovery works within and across peers
- [Architecture](dir-architecture.md) — Directory components and trust boundaries
