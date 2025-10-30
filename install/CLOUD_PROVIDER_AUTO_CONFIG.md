# Cloud Provider Auto-Configuration for Routing Service

## Overview

The dir-server Helm chart now supports **automatic cloud provider configuration** for the routing service. Simply specify your cloud provider, and the appropriate LoadBalancer annotations are automatically applied!

## Features

✅ **Automatic annotation generation** for AWS, GCP, and Azure  
✅ **Provider-specific options** for advanced configuration  
✅ **Manual annotation override** - custom annotations take precedence  
✅ **Backward compatible** - existing manual configurations still work  
✅ **Zero maintenance** - annotations stay up-to-date with provider best practices  
✅ **Local-friendly default** - uses NodePort by default (works in Kind, Minikube, and cloud)  

## Quick Start

> **Note:** The default service type is `NodePort` (works everywhere). For production cloud deployments, explicitly set `type: LoadBalancer` to get a stable external IP.

### AWS Deployment

```yaml
# values.yaml or via --set
apiserver:
  routingService:
    type: LoadBalancer  # Change from default NodePort to LoadBalancer
    cloudProvider: "aws"
    externalTrafficPolicy: Local
```

**Automatically generates:**
```yaml
annotations:
  service.beta.kubernetes.io/aws-load-balancer-type: "nlb"
  service.beta.kubernetes.io/aws-load-balancer-scheme: "internet-facing"
  service.beta.kubernetes.io/aws-load-balancer-cross-zone-load-balancing-enabled: "true"
```

### GCP Deployment

```yaml
apiserver:
  routingService:
    type: LoadBalancer
    cloudProvider: "gcp"
    externalTrafficPolicy: Local
```

**Automatically generates:**
```yaml
annotations:
  cloud.google.com/load-balancer-type: "External"
```

### Azure Deployment

```yaml
apiserver:
  routingService:
    type: LoadBalancer
    cloudProvider: "azure"
    externalTrafficPolicy: Local
```

**Automatically generates:**
```yaml
annotations:
  service.beta.kubernetes.io/azure-load-balancer-internal: "false"
```

## Advanced Configuration

### AWS-Specific Options

```yaml
apiserver:
  routingService:
    type: LoadBalancer
    cloudProvider: "aws"
    aws:
      internal: true  # Use internal NLB instead of internet-facing
      nlbTargetType: "ip"  # Use IP-based targets (default: instance)
```

**Generated annotations:**
```yaml
annotations:
  service.beta.kubernetes.io/aws-load-balancer-type: "nlb"
  service.beta.kubernetes.io/aws-load-balancer-scheme: "internal"  # ← Changed
  service.beta.kubernetes.io/aws-load-balancer-internal: "true"  # ← Added
  service.beta.kubernetes.io/aws-load-balancer-cross-zone-load-balancing-enabled: "true"
  service.beta.kubernetes.io/aws-load-balancer-nlb-target-type: "ip"  # ← Added
```

### GCP-Specific Options

```yaml
apiserver:
  routingService:
    type: LoadBalancer
    cloudProvider: "gcp"
    gcp:
      internal: true  # Use internal load balancer
      backendConfig: "my-backend-config"  # Custom BackendConfig
```

**Generated annotations:**
```yaml
annotations:
  cloud.google.com/load-balancer-type: "Internal"  # ← Changed
  cloud.google.com/backend-config: "my-backend-config"  # ← Added
```

### Azure-Specific Options

```yaml
apiserver:
  routingService:
    type: LoadBalancer
    cloudProvider: "azure"
    azure:
      internal: true  # Use internal load balancer
      resourceGroup: "my-resource-group"  # Custom resource group
```

**Generated annotations:**
```yaml
annotations:
  service.beta.kubernetes.io/azure-load-balancer-internal: "true"  # ← Changed
  service.beta.kubernetes.io/azure-load-balancer-resource-group: "my-resource-group"  # ← Added
```

## Manual Annotation Override

Custom annotations **always take precedence** over auto-generated ones:

```yaml
apiserver:
  routingService:
    type: LoadBalancer
    cloudProvider: "aws"  # Auto-generates AWS annotations
    annotations:
      # Override the scheme annotation
      service.beta.kubernetes.io/aws-load-balancer-scheme: "internal"
      # Add custom annotation
      custom.io/my-annotation: "custom-value"
```

**Result:**
```yaml
annotations:
  # Auto-generated (not overridden)
  service.beta.kubernetes.io/aws-load-balancer-type: "nlb"
  service.beta.kubernetes.io/aws-load-balancer-cross-zone-load-balancing-enabled: "true"
  # Manually overridden
  service.beta.kubernetes.io/aws-load-balancer-scheme: "internal"
  # Custom annotation
  custom.io/my-annotation: "custom-value"
```

## Configuration Precedence

The annotation resolution follows this order:

1. **Manual annotations** (`routingService.annotations`) - highest priority
2. **Auto-generated provider annotations** (`cloudProvider` + provider-specific options)
3. **No annotations** (if neither is set)

## Deployment Examples

### Via Helm CLI (AWS)

```bash
helm upgrade --install dir ./install/charts/dir \
  --set apiserver.routingService.type=LoadBalancer \
  --set apiserver.routingService.cloudProvider=aws \
  --set apiserver.routingService.externalTrafficPolicy=Local \
  -n dir-server --create-namespace
```

### Via Helm CLI (GCP with static IP)

```bash
# Reserve static IP first in GCP console
# gcloud compute addresses create dir-routing-ip --region=us-central1

helm upgrade --install dir ./install/charts/dir \
  --set apiserver.routingService.type=LoadBalancer \
  --set apiserver.routingService.cloudProvider=gcp \
  --set apiserver.routingService.loadBalancerIP=35.123.45.67 \
  --set apiserver.routingService.externalTrafficPolicy=Local \
  -n dir-server --create-namespace
```

### Via Values File (Azure internal)

```yaml
# custom-values.yaml
apiserver:
  service:
    type: ClusterIP
  
  routingService:
    type: LoadBalancer
    cloudProvider: "azure"
    externalTrafficPolicy: Local
    azure:
      internal: true
      resourceGroup: "my-resource-group"
```

Deploy:
```bash
helm upgrade --install dir ./install/charts/dir \
  -f custom-values.yaml \
  -n dir-server --create-namespace
```

## Testing Template Rendering

Verify your configuration before deploying:

```bash
# Test AWS configuration
helm template test ./install/charts/dir \
  --set apiserver.config.routing.listen_address="/ip4/0.0.0.0/tcp/5555" \
  --set apiserver.routingService.type=LoadBalancer \
  --set apiserver.routingService.cloudProvider=aws \
  | grep -A 30 "kind: Service" | grep -B5 -A25 routing

# Test GCP configuration
helm template test ./install/charts/dir \
  --set apiserver.config.routing.listen_address="/ip4/0.0.0.0/tcp/5555" \
  --set apiserver.routingService.cloudProvider=gcp \
  | grep -A 30 "apiserver-routing"

# Test Azure configuration  
helm template test ./install/charts/dir \
  --set apiserver.config.routing.listen_address="/ip4/0.0.0.0/tcp/5555" \
  --set apiserver.routingService.cloudProvider=azure \
  | grep -A 30 "apiserver-routing"
```

## Migration from Manual Configuration

If you're currently using manual annotations:

### Before (manual):
```yaml
apiserver:
  routingService:
    type: LoadBalancer
    annotations:
      service.beta.kubernetes.io/aws-load-balancer-type: "nlb"
      service.beta.kubernetes.io/aws-load-balancer-scheme: "internet-facing"
      service.beta.kubernetes.io/aws-load-balancer-cross-zone-load-balancing-enabled: "true"
```

### After (automatic):
```yaml
apiserver:
  routingService:
    type: LoadBalancer
    cloudProvider: "aws"  # Much simpler!
```

**Benefits of migration:**
- ✅ Less configuration to maintain
- ✅ Automatic updates to best practices
- ✅ Consistent across deployments
- ✅ Still supports custom annotations when needed

## Backward Compatibility

Existing deployments with manual annotations continue to work without changes:

```yaml
# This still works exactly as before
apiserver:
  routingService:
    type: LoadBalancer
    annotations:
      my.custom/annotation: "value"
```

## Implementation Details

The automatic configuration is implemented via a Helm template helper:

**Location:** `/install/charts/dir/apiserver/templates/_helpers.tpl`

**Template:** `chart.routingService.annotations`

**Logic:**
1. Check if `cloudProvider` is set
2. Generate provider-specific annotations based on provider and options
3. Merge with manual `annotations` (manual takes precedence)
4. Output final annotation dictionary

This approach ensures:
- Clean separation of concerns
- Easy to extend with new providers
- Testable via `helm template`
- No runtime dependencies

## Troubleshooting

### Annotations not appearing

**Check template rendering:**
```bash
helm template test ./install/charts/dir \
  --set apiserver.config.routing.listen_address="/ip4/0.0.0.0/tcp/5555" \
  --set apiserver.routingService.cloudProvider=aws \
  --debug
```

### Wrong provider annotations

**Verify cloudProvider value:**
- Must be exactly: `"aws"`, `"gcp"`, or `"azure"` (lowercase)
- Empty string = no auto-configuration

### LoadBalancer pending

```bash
kubectl describe svc dir-apiserver-routing -n dir-server
```

Common causes:
- Cloud provider doesn't support LoadBalancer (local cluster)
- Insufficient permissions
- Quota exceeded
- Check Events section in describe output

## Best Practices

1. **Use auto-configuration for standard deployments**
   - Simplifies configuration
   - Reduces errors
   - Stays current with best practices

2. **Use manual annotations for edge cases**
   - Custom requirements
   - Preview features
   - Provider-specific advanced features

3. **Combine both when needed**
   - Let auto-config handle basics
   - Override specific settings manually
   - Add custom annotations

4. **Test before deploying**
   - Use `helm template` to verify
   - Check annotations in output
   - Validate against provider requirements

## Summary

The cloud provider auto-configuration feature simplifies multi-cloud deployments while maintaining flexibility for advanced use cases. Choose the approach that fits your needs:

| Use Case | Approach |
|----------|----------|
| **Standard AWS/GCP/Azure** | ✅ Use `cloudProvider` auto-config |
| **Custom requirements** | Use manual `annotations` |
| **Mix of both** | Combine auto-config + manual overrides |
| **Existing manual config** | Keep as-is (backward compatible) |

This ensures your dir-server routing service is properly configured for production P2P networking across any cloud provider! 🚀

