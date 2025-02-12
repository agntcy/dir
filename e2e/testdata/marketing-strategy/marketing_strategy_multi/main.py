# SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
# SPDX-License-Identifier: Apache-2.0

import os
import threading
import traceback

from dotenv import load_dotenv
from marketing_strategy_multi.runtime.observability.observer import Observer
from marketing_strategy_multi.runtime.observability.serve import EventServer, GraphDashboard
from marketing_strategy_multi.utils.parser import parse_args, parse_inputs
from marketing_strategy_multi.utils.state import MarketingStrategyState
from marketing_strategy_multi.workflow import WorkFlow


def run():
    load_dotenv()
    dashboard_enabled = os.getenv('DASHBOARD_ENABLED', 'True').lower() == 'true'
    http_host = os.getenv('HTTP_HOST', '127.0.0.1')
    http_port = int(os.getenv('HTTP_PORT', 8050))
    http_debug = os.getenv('HTTP_DEBUG', 'True').lower() == 'true'
    
    args = parse_args()
    start_state: MarketingStrategyState = {"state": parse_inputs(args)}
    
    observer = Observer()
    workflow = WorkFlow(observers=[observer])
    observer.graph = workflow.get_graph()
    workflow_thread = threading.Thread(target=workflow.run, args=(start_state,))
    workflow_thread.start()
    
    try:
        eventserver = EventServer(observer)

        if dashboard_enabled:
            dashboard = GraphDashboard(observer, eventserver)
            # Debug need to be false because somehow it triggers the graph execution again. (I tried to disable the hot reload function but it was unreflective)
            dashboard.run(http_host, http_port, debug=False)
        else:
            eventserver.run(http_host, http_port, http_debug)

    except Exception as e:
        print(f"An error occurred: {e}")
        traceback.print_exc()

    finally:
        workflow_thread.join()
    
if __name__ == "__main__":
    run()
