# ADS Third-Party Integration Options

## Overview

This documents outlines research details of ADS integration support options with third-party services.

## Goal

- Minimal or no changes required on ADS and OASF projects
- Enable simple integration path of AGNTCY components
- Leverage existing and widely-adopted tooling for agentic development

## Methodology

All workflows try encapsulate three important aspecs in order to support this goal.

- **Schema Extensions** - Focus only on the data, its contents and structure, e.g. LLMs, A2A, MCP servers. Use the findings to provide an example OASF record, given as `record.json`
- **Data Extractors and Transformers** - Provide logic that reads, extracts, and transforms the data into service-specific artifacts that can be used with existing services (ie. VSCode Copilot MCP server support).
Use OASF records as a data carriers.
- **Usable and Useful Workflows** - The third-party tools can be easily configured and used.

## Steps taken

The integration support was carried out in the following way:

1. Gather common agentic workflows used by AI developers. Outcome: *devs mainly use plain LLMs with MCP servers*.
2. Gather common tools used by AI developers. Outcome: *devs mainly use IDEs (VSCode) with mentioned workflows*.
3. Attach common agentic data to OASF records. Settle for **LLMs, MCP servers, and A2A card details**.
4. Provide a script that uses data from 3. to support 1. and 2.

Focus on the following integrations in the initial PoC:

- **VSCode Copilot in Agent Mode** -- only supports MCP server configuration
- **Continue.dev VSCode extension** -- supports prompts, LLMs, and MCP server + extra

## Outcome

The data around LLM, MCP, and A2A can be easily added to existing OASF schema via extension fields.
This can be verified via `record.json` file.
If needed, these extensions can also be moved as first-class schema properties, which is also easily supported by OASF.

The data extranction and transformation logic can be easily added, either as standalone scripts, or as part of the directory client.
This can be verified via `poc.py` script.
If needed, extractor/transformer interface can be used on the `dirctl` CLI for different tools which can be easily implemented given the list of integrations to support.

In summary, this demonstrates the usage of OASF and ADS to easily add out-of-box support for third-party tools and services.

## Usage

### Requirements

- *VSC Copilot-based workflow*: Login to Copilot from VSCode
- *Continue-based workflow*: Install https://www.continue.dev/ VSCode extension
- *Python3 with virtual environment installed*

### Steps

1. Run `task poc:integration` - This step generates artifacts for both workflow-types (VSC + Continue).
   It saves them to workflow-specific directory for the current VS Code project, ie. `.vscode/` and `.continue/assistants`.

2. Run `cp docs/poc/integrations/.env.example .env` - This step sets up ENV-var inputs for Continue-based workflow. Fill the env vars after setup.
   This is required for Continue as it does not support prompt inputs.
   VSC Copilot will ask for all the necessary inputs via prompts when we start interacting with it.

3. *VSC Copilot-based workflow* - Open the chat console. Switch to LLM such as Claude. Switch to Agent mode.

4. *Continue-based workflow* - Open the chat console. Refresh the Assistants tab to load our Assistant created from OASF record. Switch to Azure GPT-4o LLM. Switch to Agent mode.

5. Run the following prompt:

```text
Summarize the pull request in detail, including its purpose, changes made, and any relevant context. Focus on the technical aspects and implications of the changes. Use the provided link to access the GitHub pull request.
Run for this PR: https://github.com/agntcy/dir/pull/179
```

This prompt will use configured MCP GitHub server to fetch the required context and will create a detailed summary about the PR.
