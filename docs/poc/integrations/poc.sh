#!/bin/bash

######################################################################
## Example using Directory as a way to integrate Agentic workflows
## with **Continue VSCode extension** natively
######################################################################

## Get the path of project root directory and the .vscode directory
SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
VSCODE_DIR=$SCRIPT_DIR/../../../.vscode

######################################################################
#################################### Hub/Directory flow
## Push record to ADS
# DIGEST=$(dirctl push record.json)

## Pull record from ADS
# dirctl pull $DIGEST > record.json

## Search record from ADS
# dirctl search --query "name:record.json" --output json > record.json

######################################################################
#################################### MCP Support
## Extract MCP information from the record
cat record.json | jq '.extensions[] | select(.name == "schema.oasf.agntcy.org/features/runtime/mcp") | .data' > mcp.json

## Add the information to VSCode
mkdir -p $VSCODE_DIR
cp -f mcp.json $VSCODE_DIR/mcp.json

## Cleanup
rm -rf mcp.json

######################################################################
#################################### Model Support

