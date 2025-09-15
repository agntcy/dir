# ADS Third-Party Integration Options

## Overview

This documents outlines research details of ADS integration support options with third-party services.

## Goal

- Minimal or no changes required on ADS and OASF projects
- Enable simple integration path of AGNTCY components
- Leverage existing and widely-adopted tooling for agentic development

## Methodology

All workflows try encapsulate three important aspecs in order to support this goal.

- **Schema Extensions** - Focus only on the data, its contents and structure, e.g. LLMs, Prompts, A2A, MCP servers. Use the findings to define required OASF Record extensions.
- **Data Extractors and Transformers** - Provide logic that reads, extracts, and transforms the data into service-specific artifacts that can be used with given services, eg. VSCode Copilot and Continue.
Use OASF records as a data carriers.
- **Usable and Useful Workflows** - Enable out-of-box configuration and usage of given services.

## Steps taken

The integration support was carried out in the following way:

1. Gather common agentic workflows used by AI developers. Outcome: *devs mainly use LLMs with MCP servers*.
2. Gather common tools used by AI developers. Outcome: *devs mainly use IDEs like VSCode Copilot*.
3. Attach common agentic data to OASF records. Settle for **LLMs, Prompts, MCP servers, and A2A card details**.
4. Provide a script that uses data from 3. to support 1. and 2.

Focus on the following integrations in the initial PoC:

- **VSCode Copilot in Agent Mode** - supports only MCP server configuration
- **Continue.dev VSCode extension** - supports LLMs, prompts, and MCP server

## Outcome

The data around LLM, Prompts, MCP, and A2A can be easily added to existing OASF schema via extensions.
This can be verified via `demo.record.json` file.
If needed, these extensions can also be moved as first-class schema properties, which is also easily supported by OASF.

The data extraction and transformation logic can be easily added, either as standalone scripts, or as part of the directory client.
This can be verified via `importer.py` script.
If needed, extractor/transformer interface can be used on the `dirctl` CLI for different tools which can be easily implemented as new plugins given the list of integrations to support.

> In summary, this demonstrates the usage of OASF and ADS to easily add out-of-box support for third-party tools and services to enable agentic development.

## Usage

### Import and configure

1. Run `task poc:integration`

This step generates artifacts for both workflow-types, including VSCode Copilot and Continue.

The artifacts are saved under workflow-specific directory for the given tool, ie. `.vscode/` and `.continue/assistants/`.

2. Run `cp docs/research/integrations/.env.example .env`

This step sets up ENV-var inputs for Continue-based workflow. Fill the env vars after setup.
This is required for Continue as it does not support prompt inputs.

VSC Copilot will ask for all the necessary inputs via prompts when the chat is started, and this step has no meaning for VS Code.

### *VSC Copilot-based workflow*
   
 1. Login to Copilot from VSCode
 2. Open the chat console
 3. Switch to LLM such as Claude
 4. Switch to Agent mode

### *Continue-based workflow*

1. Open the chat console
2. Refresh the Assistants tab
3. Switch to our OASF-based assistant
4. Switch to Azure GPT-4o LLM
5. Switch to Agent mode.

### Try it out with a prompt

```text
Summarize the pull request in detail, including its purpose, changes made, and any relevant context. Focus on the technical aspects and implications of the changes. Use the provided link to access the GitHub pull request.
Run for this PR: https://github.com/agntcy/dir/pull/179
```

This prompt will use configured MCP GitHub server to fetch the required context and will create a detailed summary about the PR.
