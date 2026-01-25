# AGNTCY Directory GUI Guide

Welcome to the **AGNTCY Directory GUI**. This application provides a visual interface for interacting with the Model Context Protocol (MCP) server, offering a streamlined experience for managing directory resources through natural language.

## Architecture: A Unified Bundle

Typically, the Model Context Protocol involves a client (like an IDE or a chat bot) talking to a separate server process. Configuring this requires managing paths, ports, and binaries manually.

**The Directory GUI simplifies this:**
- **Embedded Server**: The application embeds the MCP Server (written in Go) directly within the app bundle.
- **Automatic Lifecycle**: When you launch the GUI, it automatically starts the MCP server in the background and establishes a secure connection.
- **Self-Contained**: You don't need to install or run a separate server command; the app handles everything.

## How it Works

Using the GUI is similar to using an AI coding assistant (like GitHub Copilot Chat), but with a specialized focus:

1.  **Chat Interface**: You type commands in natural language (e.g., "List all agents," "Register a new service," "Find events related to X").
2.  **Tool Execution**: The LLM interprets your request and decides to call specific **MCP Tools**.
3.  **Visualization**: instead of just seeing text or JSON, the GUI can render specialized views for the data returned by the tools, providing a richer experience than a standard terminal.

It bridges the gap between raw CLI commands and high-level intent, allowing you to "chat" with your system architecture.

## Settings & Configuration

The **Settings Screen** allows you to configure the "Brain" of the application (the LLM) and the connections to external services.

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
