# Federation Best Practices and Troubleshooting

This page covers operational best practices and troubleshooting for Directory federation. It applies to both [https_web and https_spiffe](dir-federation-profiles.md#profile-comparison) profiles. For profile selection and configuration details, see [Federation Bundle Profiles](dir-federation-profiles.md).

## Operational Best Practices

### Universal Best Practices

- Consistent className configuration

    - Use standardized className value across all resources (recommended: "dir-spire")
    - Ensures SPIRE Controller Manager correctly processes ClusterSPIFFEID resources
    - Prevents controller filtering issues

- Enable ClusterFederatedTrustDomain Controller

    - Required for automatic trust bundle fetching
    - Without this controller, federation configuration is ignored
    - Verify in controller manager logs

- Implement Bundle Refresh Monitoring

    - Monitor SPIRE server logs for "Bundle refreshed" events
    - Alert on bundle staleness exceeding 24 hours
    - Implement automated monitoring:

      ```bash
      kubectl logs -n dir-spire -l app.kubernetes.io/name=server -c spire-server | \
        grep "Bundle refreshed" | \
        grep "trust_domain=partner.com"
      ```

- Configure Rate Limiting

    - Protect federation endpoints from abuse
    - Recommended configuration: 100 requests/second with 5x burst multiplier
    - Adjust based on number of federation partners and refresh frequency

- Set Appropriate CA TTL

    - Recommended: 720 hours (30 days)
    - Balances security (shorter TTL) with operational stability (fewer rotations)
    - Consider operational capacity for handling rotation issues

### https_web Specific Practices

- Use Production Certificate Authority

    - Deploy with production Let's Encrypt (`letsencrypt` ClusterIssuer)
    - Avoid staging certificates in production federation scenarios
    - Staging certificates are not trusted by remote SPIRE servers by default

- Monitor Certificate Expiration

    - Although cert-manager auto-renews, implement monitoring
    - Alert on certificates expiring within 7 days
    - Verify renewal process during certificate lifecycle:

      ```bash
      kubectl get certificate -n dir-spire spire-server-federation-cert -o yaml | \
        grep -A5 "status:"
      ```

- Validate External Network Accessibility

    - Periodically test federation endpoint from external network
    - Verify DNS resolution from federation partner networks
    - Validate certificate chain visibility:

      ```bash
      curl -v https://spire.example.com 2>&1 | grep -A10 "SSL certificate"
      ```

### https_spiffe Specific Practices

- Maintain Bootstrap Bundle Freshness

    - Update bootstrap bundles before CA rotation (every caTTL period)
    - Implement bundle distribution automation where feasible
    - Document manual update procedures for operational staff

- Document Bundle Exchange Procedures

    - Establish clear onboarding process for new federation partners
    - Define secure channels for bundle transmission
    - Maintain contact registry for federation partner technical contacts

- Validate SSL Passthrough Persistence

    - Verify ssl-passthrough configuration after cluster maintenance
    - Test after NGINX ingress controller upgrades
    - Automate validation:
    
      ```bash
      kubectl get deployment -n ingress-nginx ingress-nginx-controller -o yaml | \
        grep "enable-ssl-passthrough" || echo "WARNING: SSL passthrough not enabled"
      ```

## Troubleshooting

### Connection Issues

```bash
# Check SPIRE agent status
spire-agent api fetch x509-svid

# Verify network connectivity
curl -v https://ads.outshift.io

# Verify env vars are set
echo $DIRECTORY_CLIENT_SERVER_ADDRESS $DIRECTORY_CLIENT_SPIFFE_SOCKET_PATH
```

### Federation Issues

```bash
# Verify trust bundle exchange
spire-server federation show --trustDomain spire.ads.outshift.io

# Test bundle endpoint
curl https://spire.ads.outshift.io/
```

### Common Errors

| Error | Solution |
|-------|----------|
| `connection refused` | Check SPIRE agent running and socket path |
| `x509: certificate signed by unknown authority` | Verify trust bundle configuration |
| `context deadline exceeded` | Check network and firewall |
| `permission denied` | Ensure SPIFFE ID registration and policies |

### https_web Profile Issues

#### Issue: "certificate is valid for ingress.local, not spire.example.com"

**Root Cause:** NGINX serving default certificate instead of cert-manager issued certificate.

**Resolution:** Ensure `tlsSecret` is set under `spire-server.ingress` in your Helm values:

```yaml
# Under spire-server.ingress in values
ingress:
  tlsSecret: "spire-server-federation-cert"
```

**Verification:**
```bash
kubectl get certificate -n dir-spire spire-server-federation-cert
# Status should be READY=True
```

#### Issue: "certificate signed by unknown authority"

**Root Cause:** cert-manager certificate not issued or ClusterIssuer misconfigured.

**Resolution:**
```bash
# Check Certificate status
kubectl describe certificate -n dir-spire spire-server-federation-cert

# Check ClusterIssuer status
kubectl get clusterissuer letsencrypt -o yaml

# Review cert-manager logs
kubectl logs -n cert-manager deployment/cert-manager
```

#### Issue: "no endpoints available for service"

**Root Cause:** Ingress not routing traffic to SPIRE server service.

**Resolution:**
```bash
# Verify Ingress status
kubectl get ingress -n dir-spire

# Check service endpoints
kubectl get endpoints -n dir-spire spire-server

# Verify SPIRE server pods are running
kubectl get pods -n dir-spire -l app.kubernetes.io/name=server
```

### https_spiffe Profile Issues

#### Issue: "certificate contains no URI SAN"

**Root Cause:** SPIRE server has not issued itself a federation X.509-SVID.

**Resolution:**
```bash
# Restart SPIRE server to trigger SVID issuance
kubectl rollout restart -n dir-spire deployment/spire-server

# Wait for rollout to complete
kubectl rollout status -n dir-spire deployment/spire-server

# Verify federation SVID issuance in logs
kubectl logs -n dir-spire -l app.kubernetes.io/name=server -c spire-server | \
  grep "federation"
```

#### Issue: "certificate signed by unknown authority"

**Root Cause:** Bootstrap bundle is missing, stale, or incorrectly formatted.

**Resolution:** Obtain a fresh bundle from your federation partner, validate it, and update `trustDomainBundle` in your federation resource. See [Bootstrap Bundle Exchange Process](dir-federation-profiles.md#bootstrap-bundle-exchange-process-https_spiffe) for extraction and validation steps:

```bash
# Obtain fresh bundle from federation partner
curl -k https://spire.partner.com > fresh-partner-bundle.json

# Validate bundle structure
jq '.' fresh-partner-bundle.json

# Update trustDomainBundle in federation configuration
# Redeploy application
```

#### Issue: "ssl-passthrough annotation not taking effect"

**Root Cause:** NGINX ingress controller not configured with `--enable-ssl-passthrough` flag.

**Resolution:**
```bash
# Verify controller has ssl-passthrough enabled
kubectl get deployment -n ingress-nginx ingress-nginx-controller -o yaml | \
  grep "enable-ssl-passthrough"

# If not present, patch deployment
kubectl patch deployment -n ingress-nginx ingress-nginx-controller --type='json' \
  -p='[{"op": "add", "path": "/spec/template/spec/containers/0/args/-", "value":"--enable-ssl-passthrough"}]'

# Verify configuration persisted after controller restart
```
