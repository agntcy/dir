# gui

A new Flutter project.

## Getting Started

This project is a starting point for a Flutter application.

A few resources to get you started if this is your first Flutter project:

- [Lab: Write your first Flutter app](https://docs.flutter.dev/get-started/codelab)
- [Cookbook: Useful Flutter samples](https://docs.flutter.dev/cookbook)

For help getting started with Flutter development, view the
[online documentation](https://docs.flutter.dev/), which offers tutorials,
samples, guidance on mobile development, and a full API reference.

## Prerequisites

Build the MCP server binary:

```bash
task mcp:build
```

## Build and Run (macOS)

Use the following command to set up the environment and run the app on macOS:

```bash
source ~/.env-testing-local && \
unset HTTP_PROXY && \
unset HTTPS_PROXY && \
export DIRECTORY_CLIENT_SERVER_ADDRESS="localhost:8888" && \
export MCP_SERVER_PATH="$PWD/../build/mcp/mcp-server" && \
export AZURE_API_KEY="$AZURE_OPENAI_API_KEY" && \
export AZURE_ENDPOINT="$AZURE_OPENAI_ENDPOINT" && \
export AZURE_DEPLOYMENT="$AZURE_OPENAI_DEPLOYMENT_NAME" && \
flutter run -d macos --no-pub
```
