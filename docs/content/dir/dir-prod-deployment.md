# Production Deployment

This guide documents production deployment of Directory on AWS EKS. For a single opinionated AWS walkthrough that carries the deployment all the way into public federation, see [Federation on Amazon EKS](dir-federation-aws-eks.md). For local development and testing, see [Quickstart](dir-quickstart.md) and [Local Deployment](dir-deployment-local.md). For connecting to the public Directory network or federating your instance, see [Federation](dir-federation-overview.md) and [Running a Federated Directory Instance](dir-federation-setup.md).

!!! important "Trust Domain Selection"
    Choose your **trust domain** carefully before deployment—it cannot be changed later. A trust domain is a permanent identifier for your SPIRE deployment (e.g., `acme.com`, `engineering.acme.com`).

    **Requirements:**
    - Must be globally unique
    - Cannot be changed after deployment
    - Does not need to be a real DNS domain (but can be)
    - Will be visible to federation partners

## Overview

Production deployment uses:

- **Platform:** AWS EKS (Elastic Kubernetes Service)
- **Ingress:** NGINX Ingress Controller with SSL passthrough
- **Identity:** SPIFFE/SPIRE for zero-trust authentication
- **Storage:** Zot OCI Registry with persistent volumes
- **GitOps:** ArgoCD for deployment management

## Architecture

```mermaid
flowchart TB
  Clients[External Clients]
  DNS[Route53 DNS]
  LB[AWS NLB<br/>Ingress 443]
  RoutingNLB[AWS NLB<br/>P2P Routing 5555]
  NGINX[NGINX Ingress Controller<br/>SSL Passthrough API / TLS Termination Zot]
  Dir[dir-apiserver SPIFFE]
  Reconciler[Reconciler]
  Zot[Zot Registry]
  PG[PostgreSQL]
  EBS[EBS Volumes]

  Clients --> DNS
  DNS --> LB --> NGINX
  DNS --> RoutingNLB -->|TCP 5555| Dir
  NGINX --> Dir
  NGINX --> Zot
  Dir --> PG
  Dir --> EBS
  Reconciler --> PG
  Reconciler --> Zot
  PG --> EBS
  Zot --> EBS
```

## Prerequisites

### Infrastructure

- **AWS EKS Cluster** – Kubernetes 1.31+
- **NGINX Ingress Controller** with `--enable-ssl-passthrough=true`
- **SPIRE** – SPIFFE Runtime Environment
- **ExternalDNS** – Automatic DNS management (Route53)
- **cert-manager** – TLS certificate management
- **External Secrets Operator** – Vault integration
- **ArgoCD** – GitOps deployment

### Network

- **Route53 Hosted Zone** – For your domains (e.g., `*.your-domain.com`)
- **AWS Network Load Balancer** – Layer 3/4 TCP passthrough for both ingress (443) and P2P routing (5555)
- **Security Groups** – Allow inbound TCP 443 (HTTPS) and TCP 5555 (P2P routing) from the internet on the load balancers; allow egress to Let's Encrypt, Route53, and federation endpoints

### Storage

- **EBS CSI Driver** – For persistent volumes
- **Storage Class** – `ebs-sc-encrypted` for production
- **Vault** – For credential storage

## Local vs Production

| Feature | Local (Kind) | Production (EKS) |
|---------|--------------|------------------|
| **Cluster** | Kind | AWS EKS |
| **SPIFFE CSI Driver** | ✅ Enabled | ✅ Enabled |
| **Storage** | emptyDir (ephemeral) | PVCs (persistent) |
| **Credentials** | Hardcoded in values | ExternalSecrets + Vault |
| **Resources** | 250m/512Mi | 500m–2000m / 1–4Gi |
| **Ingress** | NodePort, port-forward | Ingress + TLS |
| **Rate Limits** | 50 RPS | 500+ RPS |
| **Trust Domain** | example.org (local only) | your-domain.com |

## Key Production Features

### SPIFFE CSI Driver

Enabled via `spire.useCSIDriver: true` (v1.0.0-rc.4+):

- Synchronous workload identity injection before pod start
- Eliminates "certificate contains no URI SAN" errors
- Required for CronJobs and short-lived workloads

### Persistent Storage

- Enable PVCs for routing datastore and database
- Use `strategy.type: Recreate` to prevent database lock conflicts
- Production example: 20Gi routing, 5Gi database, 100Gi Zot

### Secure Credential Management

- Use External Secrets Operator with Vault
- See [External Secrets Operator documentation](https://external-secrets.io/latest/)

### SSL Passthrough for DIR API

The Directory API uses SPIFFE mTLS. Ingress must:

1. **Not** terminate TLS
2. Pass encrypted traffic to the backend
3. Route based on SNI

**Ingress configuration:**

```yaml
annotations:
  nginx.ingress.kubernetes.io/ssl-passthrough: "true"
  nginx.ingress.kubernetes.io/backend-protocol: "GRPCS"

tls:
  - hosts:
      - api.your-domain.com
    # NO secretName - required for SSL passthrough!
```

**NGINX Ingress Controller** must have `--enable-ssl-passthrough=true`.

## DNS Hostnames

Create DNS records for your domain. Example with `your-domain.com`:

| Service | Hostname | Port |
|---------|----------|------|
| **Directory API** | api.your-domain.com | 443 (SSL passthrough) |
| **Zot Registry** | zot.your-domain.com | 443 (TLS termination) |
| **P2P Routing** | routing.your-domain.com | 5555 (TCP via NLB) |
| **SPIRE Federation** | spire.your-domain.com | 443 (TLS termination) |
| **SPIRE OIDC** | oidc-discovery.spire.your-domain.com | 443 (TLS termination) |

If you also want authenticated access for external users, `dirctl`, or automation, pair the production deployment with the optional OIDC gateway pattern described in [OIDC Authentication for Directory](dir-component-oidc-authentication.md). This is separate from SPIRE OIDC discovery used for federation.

## Verification

### SSL Passthrough

```bash
# Should show SPIFFE certificate, not "ingress.local"
echo | openssl s_client -connect api.your-domain.com:443 \
  -servername api.your-domain.com 2>/dev/null | \
  openssl x509 -noout -subject

# Expected: C=US, O=SPIRE, CN=api.your-domain.com (or your trust domain)
```

### SPIFFE Authentication

```bash
kubectl logs -n <your-dir-namespace> -l app.kubernetes.io/name=apiserver | \
  grep "Successfully obtained valid X509-SVID"
```

### P2P Routing Service

```bash
# Verify the NLB and DNS record exist
dig +short routing.your-domain.com

# Verify TCP connectivity on port 5555
nc -zv routing.your-domain.com 5555

# Check the apiserver logs for the published multiaddr
kubectl logs -n <your-dir-namespace> -l app.kubernetes.io/name=apiserver | \
  grep "multiaddr"
```

If the routing service is unreachable, peer discovery and publication will fail silently while the rest of the deployment appears healthy.

### CronJobs

```bash
kubectl get pods -n <your-dir-admin-namespace> --sort-by=.metadata.creationTimestamp | tail -10
```

## Troubleshooting

### "certificate contains no URI SAN"

- Verify SSL passthrough is working (certificate test above)
- Ensure `useCSIDriver: true` in values
- Check SPIRE entry has synced

### "certificate is valid for ingress.local"

- DNS may point to wrong LoadBalancer
- Ensure Ingress TLS section has `hosts` but **no** `secretName`
- Verify NGINX has `--enable-ssl-passthrough=true`

### Peer Discovery or Publication Fails Silently

- Verify `routing.your-domain.com` resolves and TCP 5555 is reachable from outside the cluster
- Check that the security group for the routing NLB allows inbound TCP 5555
- Confirm ExternalDNS created the routing DNS record: `kubectl get svc -n <your-dir-namespace>` should show an `EXTERNAL-IP` for the routing `LoadBalancer` service
- Check apiserver logs for multiaddr registration errors

### ConfigMap Changes Not Taking Effect

ConfigMaps are mounted at pod creation. Restart the deployment:

```bash
kubectl rollout restart deployment/<your-apiserver-deployment> -n <your-dir-namespace>
```

## Reference

- [dir-staging](https://github.com/agntcy/dir-staging) – Example deployment with ArgoCD and SPIRE (uses `ads.outshift.io` and related hosts for the public Directory)
- [OIDC Authentication for Directory](dir-component-oidc-authentication.md) – External OIDC auth model, IdP options, and edge authorization flow
- [Running a Federated Directory Instance](dir-federation-setup.md) – Federation setup for connecting to the public network
- [Federation Profiles](dir-federation-profiles.md) – Profile comparison and configuration
