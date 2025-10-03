#!/bin/sh
# Test if the personal-chain container is running and responding to JSON-RPC

set -e

# Start the container in the background
docker run -d --rm -p 8545:8545 --name test-personal-chain personal-chain > container_id.txt

# Wait for the node to be ready (max 10s)
for i in $(seq 1 10); do
  if curl -s http://localhost:8545 > /dev/null; then
    break
  fi
  sleep 1
done

# Make a JSON-RPC call to eth_blockNumber
RESPONSE=$(curl -s -X POST --data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' -H "Content-Type: application/json" http://localhost:8545)

# Print the response
echo "eth_blockNumber response: $RESPONSE"

# Stop the container
docker stop test-personal-chain
rm -f container_id.txt

# Check if the response contains a result
if echo "$RESPONSE" | grep -q '"result"'; then
  echo "Test passed: Chain is running and responding."
  exit 0
else
  echo "Test failed: No result in response."
  exit 1
fi
