// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.24;

// Agent details
struct Agent {
	string id;
	string signature;
	address owner;
}

// AgentStore definition
contract AgentStore {
	// Store params
	uint256 private total_agents;
	mapping(string => Agent) private agents;

	// Events
	event Added(Agent agent, uint256 timestamp);

	// Methods
	function add(Agent calldata req) public {
		// Verify
		require(bytes(agents[req.id].id).length == 0, "Agent already verified");

		// Create agent
		Agent memory agent = Agent({
			id: req.id,
			signature: req.signature,
			owner:msg.sender
		});

		// Store
		agents[req.id] = agent;
		total_agents++;

		// Notify
		emit Added(agent, block.timestamp);
	}

	function get(string calldata agent_id) public view returns(Agent memory) {
		return agents[agent_id];
	}

	function total() public view returns(uint256) {
		return total_agents;
	}
}
