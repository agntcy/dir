# Testing Guide

## Quick Start

```bash
cd auth/cmd/envoy-authz

# Start all services
docker-compose up --build

# In another terminal, run tests
export GITHUB_TOKEN=gho_your_oauth_token_here
./test/test.sh
```

## What Gets Tested

1. ✅ Health check (no auth)
2. ✅ Request without auth → 401
3. ✅ Request with invalid token → 401
4. ✅ Request with valid GitHub OAuth2 token → 200
5. ✅ User info headers forwarded

## Services

- **envoy-authz** (port 9002) - Our ExtAuthz service
- **envoy** (port 8080) - Envoy gateway with ext_authz filter
- **mock-directory** (port 8888) - Mock backend (echoes headers)

## Testing Manually

```bash
# Valid request with OAuth2 token
curl -H "Authorization: Bearer $GITHUB_TOKEN" \
     http://localhost:8080/api/test | jq .

# Check logs
docker-compose logs envoy-authz
docker-compose logs envoy
docker-compose logs mock-directory

# Envoy admin
curl http://localhost:9901/stats | grep ext_authz
```

## Getting a GitHub OAuth2 Token

To obtain a GitHub OAuth2 token for testing:

```bash
# Use dirctl to authenticate and get a token
dirctl auth login
dirctl auth status  # Shows your current token

# Or use GitHub CLI
gh auth token
```

## Configuration

Edit `docker-compose.yml` environment variables:

```yaml
ALLOWED_ORG_CONSTRUCTS: "agntcy,spiffe"  # Restrict to these orgs
USER_DENY_LIST: "blocked-user"           # Block specific users
```

## Cleanup

```bash
docker-compose down
```
