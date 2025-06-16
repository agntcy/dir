#!/bin/bash

## Example using Directory as a way to distribute MCP information
##
## This POC can be used with **VSCode Copilot Agent mode** natively

## Push record to ADS
# DIGEST=$(dirctl push record.json)

## Pull record from ADS
# dirctl pull $DIGEST > record.json

## Extract MCP information from the record
cat record.json | jq '.extensions[] | select(.name == "mcp-server") | .data' > mcp.json

## Add the MCP information to VSCode
mv mcp.json ./../../../.vscode
