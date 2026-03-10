# Testing Guide

## Quick Start

```bash
cd auth/cmd/envoy-authz

# Start all services
docker-compose up --build

# In another terminal, run tests
./test/test.sh
```

## What Gets Tested

1. ✅ Health check (no auth)
2. ✅ Request without x-jwt-payload → 401
3. ✅ Request with invalid payload → 401
4. ✅ Request with valid OIDC payload → 200
5. ✅ Principal headers forwarded (x-authorized-principal, x-user-id, x-principal-type)

## Services

- **envoy-authz** (port 9002) - OIDC ExtAuthz service (reads x-jwt-payload, Casbin RBAC)
- **envoy** (port 8080) - Envoy gateway with ext_authz filter
- **mock-directory** (port 8888) - Mock backend (echoes headers)

## Testing Manually

```bash
# Valid request with mock JWT payload (dev/test only)
curl -H "x-jwt-payload: {\"iss\":\"https://tenant.zitadel.cloud\",\"sub\":\"77776025198584418\"}" \
     http://localhost:8080/api/test | jq .

# Check logs
docker-compose logs envoy-authz
docker-compose logs envoy
docker-compose logs mock-directory

# Envoy admin
curl http://localhost:9901/stats | grep ext_authz
```

## Configuration

The test uses `test/config.test.yaml` mounted at `/etc/envoy-authz/config.yaml`. It allows:

- **Public path**: `/healthz` (no auth)
- **Admin user**: `user:https://tenant.zitadel.cloud:77776025198584418` (all methods)

To test deny list or different roles, edit `test/config.test.yaml` and restart:

```yaml
userDenyList:
  - "user:https://tenant.zitadel.cloud:blocked-sub"

roles:
  admin:
    allowedMethods: ["*"]
    users:
      - "user:https://tenant.zitadel.cloud:77776025198584418"
```

## Production Flow

In production, Envoy's `jwt_authn` filter validates the Bearer JWT and sets `x-jwt-payload` before ext_authz. This test setup passes `x-jwt-payload` directly for simplicity (dev/test only).

## Cleanup

```bash
docker-compose down
```
