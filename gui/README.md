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

1. **Flutter SDK** - Install Flutter from [flutter.dev](https://flutter.dev)
   - Verify installation: `flutter doctor`

2. **macOS Requirements** (for macOS development):
   - **Xcode** - Full Xcode.app (not just Command Line Tools)
     - Install from the App Store
     - After installation, run:
       ```bash
       sudo xcode-select --switch /Applications/Xcode.app/Contents/Developer
       sudo xcodebuild -runFirstLaunch
       ```
   - **CocoaPods** - Install if not already present:
     ```bash
     brew install cocoapods
     # or
     sudo gem install cocoapods
     ```

3. **macOS Sandbox Configuration** (required to execute external binaries):
   - The app needs to execute the MCP server binary, which requires disabling the App Sandbox for debug builds
   - **Check the configuration**: Open `macos/Runner/DebugProfile.entitlements` and verify it contains:
     ```xml
     <key>com.apple.security.app-sandbox</key>
     <false/>
     ```
   - **If not configured or you get "Operation not permitted" errors**:
     1. Open `macos/Runner/DebugProfile.entitlements`
     2. Ensure `com.apple.security.app-sandbox` is set to `false` (not `true`)
     3. The file should look like this:
        ```xml
        <?xml version="1.0" encoding="UTF-8"?>
        <!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
        <plist version="1.0">
        <dict>
        	<key>com.apple.security.app-sandbox</key>
        	<false/>
        	<key>com.apple.security.cs.allow-jit</key>
        	<true/>
        	<key>com.apple.security.network.server</key>
        	<true/>
        </dict>
        </plist>
        ```
     4. After making changes, run `flutter clean` and rebuild the app

4. **Build the MCP server binary**:
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
export MCP_SERVER_PATH="$PWD/../bin/mcp-server" && \
export OASF_API_VALIDATION_SCHEMA_URL="${OASF_API_VALIDATION_SCHEMA_URL:-https://schema.oasf.outshift.com}" && \
export AZURE_API_KEY="$AZURE_OPENAI_API_KEY" && \
export AZURE_ENDPOINT="$AZURE_OPENAI_ENDPOINT" && \
export AZURE_DEPLOYMENT="$AZURE_OPENAI_DEPLOYMENT_NAME" && \
flutter run -d macos --no-pub
```

## Running Tests

To run the unit and widget tests:

```bash
flutter test
```

To run the MCP integration tests (which require the built MCP server binary):

1. Ensure the MCP server is built. From the project root run:
   ```bash
   task mcp:build
   ```
2. Run the tests:
   ```bash
   flutter test
   ```
