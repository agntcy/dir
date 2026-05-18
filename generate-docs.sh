#!/bin/bash

# Generate markdown documentation from proto files using protoc-gen-doc
# This script generates documentation for all proto files in the api/proto directory

set -e

# Create docs directory if it doesn't exist
mkdir -p docs

# Find all proto files
PROTO_FILES=$(find proto -name "*.proto" -type f)

echo "Found proto files:"
echo "$PROTO_FILES"
echo ""

# Generate markdown documentation
echo "Generating markdown documentation..."
protoc --proto_path=proto --doc_out=docs --doc_opt=markdown,api-documentation.md $PROTO_FILES

echo "Documentation generated successfully!"
echo "Output file: docs/api-documentation.md" 