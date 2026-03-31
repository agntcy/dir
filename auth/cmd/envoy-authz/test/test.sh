#!/bin/bash
# Test script for OIDC ext_authz integration.
# Sends x-jwt-payload with mock JWT claims (dev/test only; production uses jwt_authn).

set -e

ENVOY_URL="http://localhost:8080"

# Mock JWT payload for testing (user in admin role per config.test.yaml)
MOCK_PAYLOAD='{"iss":"https://dex.example.com","email":"admin@example.com"}'

echo "🧪 Testing OIDC ext_authz Integration"
echo "====================================="
echo ""

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Test 1: Health check (no auth)
echo "Test 1: Health check (no auth required)"
echo "----------------------------------------"
curl -s "$ENVOY_URL/healthz" | jq .
echo -e "${GREEN}✓ Health check passed${NC}\n"

# Test 2: Request without x-jwt-payload (should fail)
echo "Test 2: Request without x-jwt-payload"
echo "--------------------------------------"
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$ENVOY_URL/api/test")
if [ "$HTTP_CODE" = "401" ]; then
    echo -e "${GREEN}✓ Correctly rejected (401 Unauthorized)${NC}\n"
else
    echo -e "${RED}✗ Expected 401, got $HTTP_CODE${NC}\n"
fi

# Test 3: Request with invalid payload (should fail)
echo "Test 3: Request with invalid x-jwt-payload"
echo "------------------------------------------"
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" \
    -H "x-jwt-payload: invalid-json" \
    "$ENVOY_URL/api/test")
if [ "$HTTP_CODE" = "401" ]; then
    echo -e "${GREEN}✓ Correctly rejected invalid payload (401)${NC}\n"
else
    echo -e "${RED}✗ Expected 401, got $HTTP_CODE${NC}\n"
fi

# Test 4: Request with valid payload (should succeed)
echo "Test 4: Request with valid x-jwt-payload"
echo "---------------------------------------"
RESPONSE=$(curl -s -w "\nHTTP_CODE:%{http_code}" \
    -H "x-jwt-payload: $MOCK_PAYLOAD" \
    "$ENVOY_URL/api/test")

HTTP_CODE=$(echo "$RESPONSE" | grep "HTTP_CODE:" | cut -d: -f2)
BODY=$(echo "$RESPONSE" | grep -v "HTTP_CODE:")

if [ "$HTTP_CODE" = "200" ]; then
    echo -e "${GREEN}✓ Request successful (200 OK)${NC}"
    echo "Response:"
    echo "$BODY" | jq .
    echo ""

    # Check OIDC headers forwarded by ext_authz
    echo "Checking forwarded principal headers..."
    PRINCIPAL=$(echo "$BODY" | jq -r '.authenticated.authorized_principal')
    USER_ID=$(echo "$BODY" | jq -r '.authenticated.user_id')
    PTYPE=$(echo "$BODY" | jq -r '.authenticated.principal_type')

    if [ "$PRINCIPAL" != "null" ] && [ "$PRINCIPAL" != "" ]; then
        echo -e "${GREEN}✓ Authorized principal: $PRINCIPAL${NC}"
    fi
    if [ "$USER_ID" != "null" ] && [ "$USER_ID" != "" ]; then
        echo -e "${GREEN}✓ User ID: $USER_ID${NC}"
    fi
    if [ "$PTYPE" != "null" ] && [ "$PTYPE" != "" ]; then
        echo -e "${GREEN}✓ Principal type: $PTYPE${NC}"
    fi
    echo ""
else
    echo -e "${RED}✗ Request failed (HTTP $HTTP_CODE)${NC}"
    echo "Response:"
    echo "$BODY" | jq .
    echo ""
fi

echo "🎉 Testing Complete!"
echo ""
echo "💡 Tips:"
echo "  - Check envoy-authz logs: docker-compose logs envoy-authz"
echo "  - Check Envoy logs: docker-compose logs envoy"
echo "  - Check mock-directory logs: docker-compose logs mock-directory"
echo "  - Envoy admin: curl localhost:9901/stats | grep ext_authz"
echo ""
