#!/bin/bash
# Test script for ext_authz integration

set -e

ENVOY_URL="http://localhost:8080"
GITHUB_PAT="${GITHUB_PAT:-}"

echo "🧪 Testing ext_authz Integration"
echo "================================="
echo ""

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test 1: Health check (no auth)
echo "Test 1: Health check (no auth required)"
echo "----------------------------------------"
curl -s "$ENVOY_URL/healthz" | jq .
echo -e "${GREEN}✓ Health check passed${NC}\n"

# Test 2: Request without auth (should fail)
echo "Test 2: Request without Authorization header"
echo "---------------------------------------------"
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$ENVOY_URL/api/test")
if [ "$HTTP_CODE" = "401" ]; then
    echo -e "${GREEN}✓ Correctly rejected (401 Unauthorized)${NC}\n"
else
    echo -e "${RED}✗ Expected 401, got $HTTP_CODE${NC}\n"
fi

# Test 3: Request with invalid token (should fail)
echo "Test 3: Request with invalid GitHub token"
echo "------------------------------------------"
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" \
    -H "Authorization: Bearer invalid_token_123" \
    "$ENVOY_URL/api/test")
if [ "$HTTP_CODE" = "401" ]; then
    echo -e "${GREEN}✓ Correctly rejected invalid token (401)${NC}\n"
else
    echo -e "${RED}✗ Expected 401, got $HTTP_CODE${NC}\n"
fi

# Test 4: Request with valid GitHub PAT (should succeed)
if [ -z "$GITHUB_PAT" ]; then
    echo -e "${YELLOW}⚠ Test 4: Skipped (set GITHUB_PAT environment variable to test)${NC}"
    echo "   Example: export GITHUB_PAT=ghp_your_token_here"
    echo ""
else
    echo "Test 4: Request with valid GitHub PAT"
    echo "--------------------------------------"
    RESPONSE=$(curl -s -w "\nHTTP_CODE:%{http_code}" \
        -H "Authorization: Bearer $GITHUB_PAT" \
        "$ENVOY_URL/api/test")
    
    HTTP_CODE=$(echo "$RESPONSE" | grep "HTTP_CODE:" | cut -d: -f2)
    BODY=$(echo "$RESPONSE" | grep -v "HTTP_CODE:")
    
    if [ "$HTTP_CODE" = "200" ]; then
        echo -e "${GREEN}✓ Request successful (200 OK)${NC}"
        echo "Response:"
        echo "$BODY" | jq .
        echo ""
        
        # Check if user info headers were added
        echo "Checking forwarded user info..."
        PROVIDER=$(echo "$BODY" | jq -r '.authenticated.provider')
        USERNAME=$(echo "$BODY" | jq -r '.authenticated.username')
        ORGS=$(echo "$BODY" | jq -r '.authenticated.org_constructs')
        
        if [ "$PROVIDER" != "null" ] && [ "$PROVIDER" != "" ]; then
            echo -e "${GREEN}✓ Provider: $PROVIDER${NC}"
        fi
        if [ "$USERNAME" != "null" ] && [ "$USERNAME" != "" ]; then
            echo -e "${GREEN}✓ Username: $USERNAME${NC}"
        fi
        if [ "$ORGS" != "null" ] && [ "$ORGS" != "" ]; then
            echo -e "${GREEN}✓ Org Constructs: $ORGS${NC}"
        fi
        echo ""
    else
        echo -e "${RED}✗ Request failed (HTTP $HTTP_CODE)${NC}"
        echo "Response:"
        echo "$BODY" | jq .
        echo ""
    fi
fi

# Test 5: Request with explicit provider header
if [ -n "$GITHUB_PAT" ]; then
    echo "Test 5: Request with explicit x-auth-provider header"
    echo "-----------------------------------------------------"
    HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" \
        -H "Authorization: Bearer $GITHUB_PAT" \
        -H "x-auth-provider: github" \
        "$ENVOY_URL/api/test")
    
    if [ "$HTTP_CODE" = "200" ]; then
        echo -e "${GREEN}✓ Request with explicit provider succeeded${NC}\n"
    else
        echo -e "${RED}✗ Request failed (HTTP $HTTP_CODE)${NC}\n"
    fi
fi

echo "🎉 Testing Complete!"
echo ""
echo "💡 Tips:"
echo "  - Check envoy-authz logs: docker-compose logs envoy-authz"
echo "  - Check Envoy logs: docker-compose logs envoy"
echo "  - Check mock-directory logs: docker-compose logs mock-directory"
echo "  - Envoy admin: curl localhost:9901/stats | grep ext_authz"
echo ""
