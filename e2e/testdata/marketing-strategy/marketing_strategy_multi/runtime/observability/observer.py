# SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
# SPDX-License-Identifier: Apache-2.0

import json
from datetime import datetime

from langgraph.graph import Graph


class Observer:
    def __init__(self,graph: Graph=None):
        self.graph = graph
        self.events = []  # List to store events as dictionaries
    
    def update(self, source, event_data=None):
        """Called when an observable source notifies the observer."""
        agent_name = event_data.get("agentName") if event_data else None
        # Identifying the predecessor of the actual agent from the graph and previous events.
        node_parents = []
        if self.graph is not None:
            node_parents = [edge.source for edge in self.graph.edges if edge.target == agent_name]

        # In langraph the first node is always __start__ as initial step.
        predecessor = '__start__'
        for event in reversed(self.events):
            # Check if the event's agent_name is in node_parents (i.e., it's a predecessor)
            # TODO currently diamond shape is not supported (we need list for predecessor for that)
            prev_event_agent_name = event['details'].get('agentName')
            if prev_event_agent_name in node_parents:
                predecessor = prev_event_agent_name
                break
            
        event = {
            "timestamp": datetime.now().isoformat(),
            "source": source,
            "details": {
                "agentName": agent_name,
                "agentPredecessor": predecessor,
                "agentPhase": event_data.get("agentPhase") if event_data else None,
                "executionID": event_data.get("executionID") if event_data else None,
                "executionPhase": event_data.get("executionPhase") if event_data else None,
                "error": event_data.get("error") if event_data else None
            }
            
        }        
           
        print(f"Observer: Update from '{source}' with event data: {event_data}")
        self.events.append(event)  # Store the event

    def get_latest_event(self):
        if self.events:
            return self.events[-1]
        return None
    
    def get_events_as_json(self):
        return json.dumps(self.events, indent=4)

class Observable:
    def __init__(self):
        self.observers = []  # List to store multiple observers

    def add_observer(self, observer):
        self.observers.append(observer)

    def remove_observer(self, observer):
        self.observers.remove(observer)

    def notify_observers(self, source, event_data=None):
        for observer in self.observers:
            observer.update(source, event_data)
