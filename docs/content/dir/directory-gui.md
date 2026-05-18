# Agent Directory GUI

Welcome to the **AGNTCY Directory GUI**. This application provides a visual
interface for interacting with the Model Context Protocol (MCP) server, offering
a streamlined experience for managing directory resources through natural
language.

## Architecture: A Unified Bundle

Typically, the Model Context Protocol involves a client (like an IDE or a chat
bot) talking to a separate server process. Configuring this requires managing
paths, ports, and binaries manually.

**The Directory GUI simplifies this:**

- **Embedded Server**: The application embeds the MCP Server (written in Go)
directly within the app bundle.
- **Automatic Lifecycle**: When you launch the GUI, it automatically starts the
MCP server in the background and establishes a secure connection.
- **Self-Contained**: You don't need to install or run a separate server
command; the app handles everything.

## How it Works

Using the GUI is similar to using an AI coding assistant (like GitHub Copilot
Chat), but with a specialized focus:

1.  **Chat Interface**: You type commands in natural language (e.g., "List all
agents," "Register a new service," "Find events related to X").
2.  **Tool Execution**: The LLM interprets your request and decides to call
specific **MCP Tools**.
3.  **Visualization**: instead of just seeing text or JSON, the GUI can render
specialized views for the data returned by the tools, providing a richer
experience than a standard terminal.

It bridges the gap between raw CLI commands and high-level intent, allowing you
to "chat" with your system architecture.

## Settings & Configuration

The **Settings Screen** allows you to configure the "Brain" of the application
(the LLM) and the connections to external services.

### LLM Providers
You can switch between different AI models to power the chat experience:

*   **Google Gemini**: Use Google's generative AI models.
*   **Azure OpenAI**: Connect to your enterprise Azure deployments.
*   **OpenAI Compatible**: Connect to standard OpenAI APIs or compatible proxies.
*   **Ollama (Local)**: Use local models running on your machine (e.g., `gemma3:4b`, `llama3`). This enables a fully local stack where data never leaves your device.

### Configuration Fields
*   **API Keys**: Securely input keys for cloud providers.
*   **Endpoints**: Custom endpoints for Azure or local servers.
*   **Model Selection**: Specify exactly which model version you want to use (e.g., defaulting to `gemma3:4b` for Ollama).

### Directory Connection
*   **Server Address**: Point the internal MCP server to your remote or local Directory Service instance.
*   **Authentication**: detailed configuration for tokens to ensure secure access to your directory data.


<video src="../assets/dir-gui.mov" controls="controls" style="max-width: 100%;"></video>

## Downloads

You can find the latest build artifacts for each platform below:

*   **Windows**: [Download Windows App](https://github.com/agntcy/dir/releases/download/gui/v1.0.0/agntcy-directory-windows.zip)
*   **macOS**: [Download macOS App](https://github.com/agntcy/dir/releases/download/gui/v1.0.0/AGNTCY-Directory.dmg)

or check the [Release Page](https://github.com/agntcy/dir/releases/tag/gui%2Fv1.0.0).

## Source Code

The source code for the GUI lives in the `gui/` directory of the [Directory
repository](https://github.com/agntcy/dir).

Build and release workflows are under [`.github/workflows/`](https://github.com/agntcy/dir/tree/main/.github/workflows) (including `gui-ci.yaml`).

## macOS Troubleshooting

macOS apps without an Apple Developer Program attestation (code signing/notarization)
can be built using Xcode, but they will be blocked by Gatekeeper upon installation,
displaying a malware warning. Users can bypass this by overriding **Privacy & Security**
settings in **System Settings** to "Open Anyway".

Alternatively, you can run the following command in your terminal to clear the quarantine attribute:

```bash
xattr -d com.apple.quarantine /Applications/AGNTCY\ Directory.app
```

## Windows Troubleshooting

The "Windows protected your PC" message (Microsoft Defender SmartScreen)
commonly appears when installing software downloaded from GitHub because the
application is unrecognized, unsigned, or lacks a high reputation score.

### How to Run the App ("Run Anyway")

To unblock an .exe file in Windows 11, right-click the file, select Properties,
check the Unblock box under the "General" tab, and click Apply. If the option is
missing, the file is not blocked.

You can also bypass SmartScreen by clicking "More info" > "Run anyway".

### Why This Happens
*   **Unknown Publisher/Unsigned Code**: Many independent developers on GitHub
   do not purchase expensive digital code-signing certificates. Windows
   defaults to blocking these.
*   **Low Reputation**: Even if signed, new apps need time for Windows to build
   a "reputation score" based on user adoption. New or rarely downloaded tools
   will trigger this warning.
*   **False Positives**: Sometimes, generic malware detection flags legitimate
   software, particularly in installer packages.


