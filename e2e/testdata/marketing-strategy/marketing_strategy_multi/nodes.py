# SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
# SPDX-License-Identifier: Apache-2.0

from marketing_strategy_multi.chief_marketing_strategist.crew import ChiefMarketingStrategistCrew
from marketing_strategy_multi.creative_content_creator.crew import CreativeContentCreatorCrew
from marketing_strategy_multi.lead_market_analyst.crew import LeadMarketAnalystCrew
from marketing_strategy_multi.runtime.nodes import Nodes
from marketing_strategy_multi.utils.crewai_telemetry import disable_crewai_telemetry
from marketing_strategy_multi.utils.telemetry import instrument

agent_attributes = {
    "lead_market_analyst": {
        "class": LeadMarketAnalystCrew,
        "program": "marketing_strategy_multi.lead_market_analyst.main",
        "name": "lead_market_analyst",
        "execution_id": "marketing-strategy-multi-v2",
        "image": "dockerhub.cisco.com/espresso-docker/tiger-team/msm:0.0.9",
        "image_pull_secrets": ["regcred"],
    },
    "chief_marketing_strategist": {
        "class": ChiefMarketingStrategistCrew,
        "program": "marketing_strategy_multi.chief_marketing_strategist.main",
        "name": "chief_marketing_strategist",
        "execution_id": "marketing-strategy-multi-v2",
        "image": "dockerhub.cisco.com/espresso-docker/tiger-team/msm:0.0.9",
        "image_pull_secrets": ["regcred"],
    },
    "creative_content_creator": {
        "class": CreativeContentCreatorCrew,
        "program": "marketing_strategy_multi.creative_content_creator.main",
        "name": "creative_content_creator",
        "execution_id": "marketing-strategy-multi-v2",
        "image": "dockerhub.cisco.com/espresso-docker/tiger-team/msm:0.0.9",
        "image_pull_secrets": ["regcred"],
    },
}

class MSMNodes(Nodes):
    def __init__(self):
        super().__init__()
        disable_crewai_telemetry()
        self.telemetry_enabled = instrument("marketing-strategy-multi")
    
    def lead_market_analyst(self, state):
        print("---> Lead Market Analyst")
        self._run_node(state, agent_attributes["lead_market_analyst"])

    def chief_marketing_strategist(self, state):
        print("---> Chief Marketing Strategist")
        self._run_node(state, agent_attributes["chief_marketing_strategist"])

    def creative_content_creator(self, state):
        print("---> Creative Content Creator")
        self._run_node(state, agent_attributes["creative_content_creator"])

