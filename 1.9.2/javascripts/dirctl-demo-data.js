/* Copyright AGNTCY Contributors (https://github.com/agntcy) */
/* SPDX-License-Identifier: Apache-2.0 */

/* Scripted demo lines and canned CLI responses for the home-page terminal. */
window.DirctlDemoData = {
  demoCid: "bafybeiexamplecidquickstartagentrecordv100demo0001",

  demoScript: [
    { type: "command", text: "dirctl daemon start" },
    { type: "output", text: "Directory daemon listening on localhost:8888" },
    { type: "pause", ms: 1500 },
    { type: "command", text: "dirctl push record.json --output raw" },
    { type: "output", text: "bafybeiexamplecidquickstartagentrecordv100demo0001" },
    { type: "pause", ms: 1500 },
    { type: "command", text: "dirctl routing publish bafybeiexamplecidquickstartagentrecordv100demo0001" },
    { type: "output", text: "Published record to routing network." },
    { type: "pause", ms: 1500 },
    {
      type: "command",
      text: 'dirctl routing search --skill "images_computer_vision" --limit 5',
    },
    {
      type: "output",
      text:
        "CID                                                          SKILL                                      PEER\n" +
        "bafybeiexamplecidquickstartagentrecordv100demo0001            images_computer_vision/image_segmentation  local\n" +
        "bafybeiexamplecidsegmentationagentv200demo0002               images_computer_vision/image_segmentation  peer-a",
    },
    { type: "pause", ms: 1500 },
    { type: "command", text: "dirctl pull bafybeiexamplecidquickstartagentrecordv100demo0001" },
    {
      type: "output",
      text:
        '{\n' +
        '  "name": "https://example.com/agents/quickstart-agent",\n' +
        '  "version": "v1.0.0",\n' +
        '  "description": "Quickstart example agent",\n' +
        '  "skills": [{ "name": "images_computer_vision/image_segmentation" }]\n' +
        "}",
    },
    { type: "pause", ms: 4000 },
  ],

  agentDemoScript: [
    { type: "user", text: "Help me triage open issues in our repository." },
    { type: "pause", ms: 1400 },
    {
      type: "agent",
      text: "I need GitHub access. Searching the Directory for an MCP server or A2A agent...",
    },
    { type: "pause", ms: 1200 },
    {
      type: "tool",
      text: 'agntcy_dir_search_local({ module: "integration/mcp", skill: "issue_tracking" })',
    },
    {
      type: "output",
      text:
        "1. github-mcp-server     MCP · stdio\n" +
        "2. issue-triage-agent    A2A · protocol 0.2.6",
    },
    { type: "pause", ms: 1400 },
    { type: "tool", text: 'mcp.add_server("github-mcp-server")' },
    { type: "output", text: "Added github-mcp-server to this session. Ready to use." },
    { type: "pause", ms: 1200 },
    { type: "user", text: "List the 5 latest open issues in acme/example" },
    { type: "pause", ms: 1000 },
    {
      type: "tool",
      text: 'github-mcp-server.list_issues({ owner: "acme", repo: "example", state: "open", limit: 5 })',
    },
    {
      type: "output",
      text:
        "#42 Improve README examples\n" +
        "#38 Add retry logic for webhooks\n" +
        "#31 Document pagination defaults\n" +
        "#24 Cache dependency lookups\n" +
        "#19 Parallelize CI workflows",
    },
    { type: "pause", ms: 4000 },
  ],

  demoTitles: {
    cli: "user@dir:~",
    agent: "agent@workspace:~",
  },

  helpText:
    "Try these commands:\n" +
    "  dirctl daemon start\n" +
    "  dirctl push record.json\n" +
    "  dirctl routing publish <cid>\n" +
    "  dirctl routing search --skill \"images_computer_vision\"\n" +
    "  dirctl pull <cid>\n" +
    "  dirctl routing list\n" +
    "  help | clear",

  dirctlHelp:
    "Directory CLI (dirctl)\n\n" +
    "Usage:\n" +
    "  dirctl [command]\n\n" +
    "Available Commands:\n" +
    "  daemon      Manage the local Directory daemon\n" +
    "  push        Store a record locally\n" +
    "  pull        Retrieve a record by CID\n" +
    "  routing     Routing operations for record discovery\n" +
    "  auth        Authentication commands\n" +
    "  help        Help about any command",

  routingHelp:
    "Routing operations for record discovery and announcement.\n\n" +
    "Available Commands:\n" +
    "  publish     Announce records to the network\n" +
    "  unpublish   Remove records from network discovery\n" +
    "  list        Query local records with filtering\n" +
    "  search      Discover remote records from other peers\n" +
    "  info        Show routing statistics",

  pullRecord:
    '{\n' +
    '  "name": "https://example.com/agents/quickstart-agent",\n' +
    '  "version": "v1.0.0",\n' +
    '  "description": "Quickstart example agent",\n' +
    '  "schema_version": "1.0.0",\n' +
    '  "skills": [\n' +
    '    { "id": 201, "name": "images_computer_vision/image_segmentation" }\n' +
    "  ],\n" +
    '  "authors": ["Quickstart"]\n' +
    "}",

  routingList:
    "CID                                                          SKILL\n" +
    "bafybeiexamplecidquickstartagentrecordv100demo0001            images_computer_vision/image_segmentation",

  routingSearch:
    "CID                                                          SKILL                                      PEER\n" +
    "bafybeiexamplecidquickstartagentrecordv100demo0001            images_computer_vision/image_segmentation  local\n" +
    "bafybeiexamplecidsegmentationagentv200demo0002               images_computer_vision/image_segmentation  peer-a",
};
