# ADS Third-Party Integration Options

## Overview

This documents outlines research details of ADS integration support options with third-party services.

## Goal

- Minimal or no changes required on ADS and OASF projects
- Enable simple integration path of AGNTCY components
- Leverage existing and widely-adopted tooling for agentic development

## Methodology

All workflows try encapsulate two important aspecs in order to support this goal.

- **Schema Extensions** - Focus only on the data, its contents and structure, e.g. LLMs, A2A, MCP servers. Use the findings to provide an example OASF record, given as `record.json`
- **Data Extractors and Transformers** - Provide logic that reads, extracts, and transforms the data into service-specific artifacts that can be used with existing services (ie. VSCode Copilot MCP server support).
Use OASF records as a data carriers.
- **Usable and Useful Workflows** - The third-party tools can be easily configured and used.

## Steps taken

The integration support was carried out in the following way:

1. Gather common agentic workflows used by AI developers. Outcome: devs mainly use plain LLMs with MCP servers.
2. Gather common tools used by AI developers. Outcome: devs mainly use IDEs (VSCode) with mentioned workflows.
3. Attach Agent-specific data to OASF records. Settle for LLM data, MCP servers, and A2A card details.
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
