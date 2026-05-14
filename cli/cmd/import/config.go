// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package importcmd

/*

enricher:
....

rules:
	- name: mcp/io.github.github
	  description: Import records from a remote MCP registry for GitHub MCPs.
	  type: mcp-registry
	  filters:
			- search=io.github.github
			- version=1.0.0
	  output:
			type: local
			local:
				output_dir: "./imported-records"

	- name: local-mcp-servers
	  description: Import records from a remote MCP registry for GitHub MCPs.
	  type: mcp
	  filters:
	  	filter: "file:./mcp-servers/**.json"
	  output:
			type: local
			local:
				output_dir: "./mcp-converted-oasf-records"

	- name: local-a2a-servers
	  description: Import records from a remote MCP registry for GitHub MCPs.
	  type: a2a
	  filters:
	  	filter: "file:./a2a-servers/**.json"
	  output:
			type: local
			local:
				output_dir: "./mcp-converted-oasf-records"

	- name: local-oasf-records
	  description: Import records from a remote MCP registry for GitHub MCPs.
	  type: oasf
	  filters:
	  	filter: "file:./servers/**.json"
	  output:
			type: remote
			remote:
				registry_address: "https://example.com/registry"

	- name: local-agent-skills
	  description: Import records from a remote MCP registry for GitHub MCPs.
	  type: oasf
	  filters:
	  	filter: "file:./servers/**.json"
	  output:
			type: remote
			remote:
				registry_address: "https://example.com/registry"

*/
