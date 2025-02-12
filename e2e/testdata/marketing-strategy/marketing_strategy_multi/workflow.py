# SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
# SPDX-License-Identifier: Apache-2.0

from typing import List

from langgraph.checkpoint.memory import MemorySaver
from langgraph.graph import StateGraph
from marketing_strategy_multi.nodes import MSMNodes, agent_attributes
from marketing_strategy_multi.runtime.observability.observer import Observer
from marketing_strategy_multi.utils.state import MarketingStrategyState


class WorkFlow:
    def __init__(self,observers: List[Observer] = None):
        self.nodes = MSMNodes()
        for observer in observers:
            self.nodes.add_observer(observer)
            
        self.workflow = StateGraph(MarketingStrategyState)

        self.workflow.add_node(agent_attributes['lead_market_analyst']['name'], self.nodes.lead_market_analyst)
        self.workflow.add_node(
            agent_attributes['chief_marketing_strategist']['name'], self.nodes.chief_marketing_strategist
        )
        self.workflow.add_node(agent_attributes['creative_content_creator']['name'], self.nodes.creative_content_creator)

        self.workflow.set_entry_point(agent_attributes['lead_market_analyst']['name'])
        self.workflow.add_edge("lead_market_analyst", "chief_marketing_strategist")
        self.workflow.add_edge("chief_marketing_strategist", "creative_content_creator")
        self.app = self.workflow.compile(
            checkpointer=MemorySaver(),
        )
    
    def get_graph(self):
        return self.app.get_graph()
    
    def run(self, start_state: MarketingStrategyState):
        ret = self.app.invoke(
            input=start_state,
            config={"configurable": {"thread_id": 10}},
        )

        print(ret["state"].outputs)
    

        
