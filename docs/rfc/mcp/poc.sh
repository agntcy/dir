#!/bin/bash

# Example poc using Directory as a way to distribute MCP information

# Push the MCP-based agent
DIGEST=$(dirctl push agent.json)

# Pull the MCP-based agent
dirctl pull $DIGEST > mcp.agent.json

# Extract MCP information from the agent
cat mcp.agent.json | jq '.extensions[] | select(.name == "mcp-server") | .data' > mcp.json

# Add the MCP information to VSCode
# mv mcp.json ~/.vscode/extensions/mcp-server/data/mcp.json
