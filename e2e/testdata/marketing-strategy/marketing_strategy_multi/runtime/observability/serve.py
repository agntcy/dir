# SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
# SPDX-License-Identifier: Apache-2.0

import json
import threading
import time
from dataclasses import dataclass, field
from typing import Any, Dict, List, NamedTuple, Optional, Union

import dash
import dash_cytoscape as cyto
from dash import dcc, html
from dash.dependencies import Input, Output, State
from flask import Flask, jsonify
from langgraph.graph import Graph
from marketing_strategy_multi.runtime.observability.observer import Observer


class EventServer:
    def __init__(self, observer):
        self.observer = observer
        self.app = Flask(__name__)
        self.setup_routes()

    def setup_routes(self):
        """Set up Flask routes."""
        @self.app.route('/events', methods=['GET'])
        def get_events():
            """Serve events in JSON format."""
            return jsonify(self.observer.events)

    def run(self, host='127.0.0.1', port=8050, debug=False, use_reloader=False):
        self.app.run(host=host, port=port, debug=debug, use_reloader=use_reloader)
        

class GraphDashboard:
    def __init__(self, observer: Observer, eventServer: EventServer = None):
        self.observer = observer
        self.graph = observer.graph
        self.active_node = '__start__'
        self.agent_phase = None
        self.active_node_predecessor = None
        self.latest_event = None
        self.lock = threading.Lock()  # Thread lock to ensure thread-safe updates
        self.dash = dash.Dash(__name__, server=eventServer.app, url_base_pathname='/dash/')
        self.setup_layout()
        self.setup_callbacks()

    def setup_layout(self):
        """Set up the layout of the Dash app."""
        stylesheet = [
            # Default Node Styles (for inactive nodes)
            {'selector': 'node', 'style': {
                'label': 'data(label)',  # Set the label
                'background-color': '#61bffc',  # Default node color (inactive)
                'width': '60px',
                'height': '60px',
                'font-size': '14px',
                'color': 'black'}},

            # Active Node Style (green when active)
            {'selector': '[nodeStatus = "active"]', 'style': {
                'background-color': '#4CAF50',  # Green for active nodes
                'width': '60px',
                'height': '60px'}},

            # Errored Node Style (red when there's an error)
            {'selector': '[nodeStatus = "error"]', 'style': {
                'background-color': '#F44336',  # Red for error nodes
                'width': '60px',
                'height': '60px'}},

            # Edge Styles
            {'selector': 'edge', 'style': {
                'line-color': '#ccc',
                'width': 2,
                'target-arrow-color': '#ccc',
                'target-arrow-shape': 'triangle',
                'curve-style': 'bezier'}},

            # Active Edge Style (for active edges)
            {'selector': '[edgeStatus = "active"]', 'style': {
                'line-color': '#4CAF50',  # Green for active edges
                'width': 4,
                'target-arrow-color': '#4CAF50'}},

            # Inactive Edge Style
            {'selector': '[edgeStatus = "inactive"]', 'style': {
                'line-color': '#2196F3',  # Blue for inactive edges
                'width': 3,
                'target-arrow-color': '#2196F3'}}
        ]

        self.dash.layout = html.Div([
            cyto.Cytoscape(
                id='cytoscape',
                elements=self.graph_to_cytoscape_elements(),  # Initial elements
                layout={'name': 'breadthfirst'},  # Define the layout
                style={'width': '100%', 'height': '500px'},
                stylesheet=stylesheet
            ),
            dcc.Interval(id='interval-component', interval=5000, n_intervals=0),  # Trigger every 5 seconds
            html.Div(id='node-info', style={'marginTop': '20px'})  # To display node information
        ])

    def graph_to_cytoscape_elements(self, active_node=None, active_node_predecessor=None, agent_phase=None):
        """Convert the graph into elements for Cytoscape."""
        elements = []
        for i, (node_key, node) in enumerate(self.graph.nodes.items()):
            node_data = {'id': node_key, 'label': node.name}
            if node_key == active_node:
                if agent_phase == "Errored":
                    node_data['nodeStatus'] = 'error'
                else:
                    node_data['nodeStatus'] = 'active'
            else:
                node_data['nodeStatus'] = 'inactive'

            elements.append({'data': node_data, 'position': {'x': 150 * i, 'y': 100 * i}})

        # Add edges to elements
        for edge in self.graph.edges:
            edge_data = {'source': edge.source, 'target': edge.target}

            # Make the edge active if it goes from active_node_predecessor to active_node
            if edge.source == active_node_predecessor and edge.target == active_node:
                edge_data['edgeStatus'] = 'active'
            else:
                edge_data['edgeStatus'] = 'inactive'

            elements.append({'data': edge_data})
        
        return elements

    def setup_callbacks(self):
        """Set up the Dash callbacks."""
        @self.dash.callback(
            Output('cytoscape', 'elements'),
            [Input('interval-component', 'n_intervals')]
        )
        def update_graph(n_intervals):
            """Update graph elements based on the current latest_event."""
            with self.lock:  # Ensure thread-safe access to shared state
                return self.graph_to_cytoscape_elements(
                    active_node=self.active_node,
                    active_node_predecessor=self.active_node_predecessor,
                    agent_phase=self.agent_phase,
                )

        @self.dash.callback(
            Output('node-info', 'children'),
            [Input('interval-component', 'n_intervals')]
        )
        def update_event_info(n_intervals):
            """Display the latest event information."""
            with self.lock:  # Ensure thread-safe access to shared state
                latest_event = self.latest_event
                if latest_event:
                    return html.Div([
                        html.H4("Latest Event"),
                        html.P(f"Time Stamp: {latest_event['timestamp']}"),
                        html.P(f"Agent Name: {latest_event['details'].get('agentName', 'N/A')}"),
                        html.P(f"Agent Phase: {latest_event['details'].get('agentPhase', 'N/A')}"),
                        html.P(f"Agent Predecessor: {latest_event['details'].get('agentPredecessor', 'N/A')}"),
                        html.P(f"Execution ID: {latest_event['details'].get('executionID', 'N/A')}"),
                        html.P(f"Execution Phase: {latest_event['details'].get('executionPhase', 'N/A')}"),
                        html.P(f"Error: {latest_event['details'].get('error', 'N/A')}")
                    ])
                return "No events to display"
        
    def handle_event(self, event_data):
        """Handle the observability event to update node states."""
        with self.lock:  # Ensure thread-safe updates
            self.active_node = event_data["details"].get("agentName")
            self.agent_phase = event_data["details"].get("agentPhase")
            self.active_node_predecessor = event_data["details"].get("agentPredecessor") or "__start__"
            self.latest_event = event_data  # Update the latest event

    def event_listener(self):
        """Poll for events and update the graph based on observability events."""
        while True:
            try:
                event_data = self.observer.get_latest_event()
                if event_data:
                    self.handle_event(event_data)
                
                time.sleep(1)
            except Exception as e:
                print(f"Error in event_listener: {e}")

    def run(self, host='127.0.0.1', port=8050, debug=True):
        listener_thread = threading.Thread(target=self.event_listener)
        listener_thread.start()
        self.dash.run(host=host, port=port, debug=debug)

