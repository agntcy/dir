# SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
# SPDX-License-Identifier: Apache-2.0

import threading
from concurrent.futures import Future, ThreadPoolExecutor

from marketing_strategy_multi.runtime.runtime import ExecutionContext, Executor


class LocalExecutor(Executor):
    def __init__(self):
        self.futures = {}
        self.executor = ThreadPoolExecutor()

    def run(
        self,
        context: ExecutionContext,
        async_mode: bool = False,
        timeout_ms: int = 60000,
    ):
        # Make sure the spec contains the necessary fields
        if "node_functions" not in context.spec:
            raise Exception("node_function not found in spec")

        # Call the nodefunction directly
        node_functions = context.spec["node_functions"]
        if async_mode:
            future = self._execute_async(node_functions[0])
            self.futures[context.metadata.name] = future
            return

        ret = node_functions[0]()

        # Save output in the status
        context.status = {"output": ret}

    def get_results(self, context: ExecutionContext) -> dict:
        if context.metadata.name not in self.futures:
            return context.status["output"]

        future = self.futures[context.metadata.name]
        return future.result()

    def _execute_async(self, node_function: callable,) -> Future[dict]:
        """Execute the task asynchronously."""
        future: Future[dict] = self.executor.submit(node_function)
        return future
