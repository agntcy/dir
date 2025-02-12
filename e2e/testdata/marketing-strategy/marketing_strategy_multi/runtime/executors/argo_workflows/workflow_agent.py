# SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
# SPDX-License-Identifier: Apache-2.0

import json
import os
import time

from kubernetes import client, config, watch
from kubernetes.dynamic import DynamicClient
from marketing_strategy_multi.runtime.executors.argo_workflows.workflow_context import WorkflowContext
from marketing_strategy_multi.runtime.executors.argo_workflows.workflow_template import workflow_template_from_file
from marketing_strategy_multi.runtime.runtime import ExecutionContext, Executor

DEBUG = os.environ.get("DEBUG", False)

class ArgoWorkflowException(Exception):
    def __init__(self, workflow_name, event, message="Argo workflow execution failed"):
        self.workflow_name = workflow_name
        self.event = event
        super().__init__(f"{message}: {workflow_name} event: {self.event}")

class ArgoExecutor(Executor):
    ARGO_API_VERSION = "argoproj.io/v1alpha1"
    ARGO_KIND = "Workflow"

    ARGO_WORKFLOW_STATUS = {
        "Pending": "Pending",
        "Running": "Running",
        "Succeeded": "Succeeded",
        "Failed": "Failed",
        "Error": "Error",
    }

    def __init__(self):
        # Create k8s client.
        if DEBUG:
            config.load_kube_config()
        else:
            config.load_incluster_config()

        self.dyn_client = DynamicClient(client.ApiClient())
        self.workflow_resource = self.dyn_client.resources.get(
            api_version=self.ARGO_API_VERSION, kind=self.ARGO_KIND
        )
        self.namespace = self._get_namespace()
        super().__init__()

    def run(
        self,
        context: ExecutionContext,
        async_mode: bool = False,
        timeout_ms: int = 60000,
    ):
        # Create a new workflow context using the provided context
        workflow_context = WorkflowContext(
            generate_name=context.metadata.name,
            image_pull_secrets=(
                context.spec["image_pull_secrets"]
                if "image_pull_secrets" in context.spec
                else []
            ),
            program=context.spec["program"],
            input_message=context.spec["input_message"],
            agent_runner_image=context.spec["agent_runner_image"],
            additional_env_vars=(
                context.spec["additional_env_vars"]
                if "additional_env_vars" in context.spec
                else {}
            ),
            additional_env_vars_from_secret=(
                context.spec["additional_env_vars_from_secret"]
                if "additional_env_vars_from_secret" in context.spec
                else {}
            ),
        )

        # Set workflow owner
        self.workflow = workflow_template_from_file(workflow_context)
        owner = self._get_workflow_owner(self.workflow)
        if owner:
            self.workflow["metadata"]["ownerReferences"] = [owner]
        # Create the Workflow
        try:
            run = self.workflow_resource.create(
                namespace=self.namespace, body=self.workflow
            )
        except Exception:
            self.notify_observers("ArgoExecutor", {
                "agentName": context.metadata.agent_name,
                "agentPhase": "Initializing",
                "executionPhase": "Error",
                "error": "Could not create Argo workflow custom resource in K8s"
            })
            raise
        
        self.notify_observers("ArgoExecutor", {
            "agentName": context.metadata.agent_name,
            "agentPhase": "Running",
            "executionID": run.metadata.name,
            "executionPhase": "Created",
            "error": None
        })

        # Save generated name in status
        context.status = {
            "workflow-name": run.metadata.name,
        }

        #  If sync, wait for it
        if not async_mode:
            # Wait for the CRD status to change to Succeeded
            ret, event = self._wait_for_crd_status(
                run.metadata.name, workflow_context.program, timeout_ms
            )
            if not ret:
                self.notify_observers("ArgoExecutor", {
                    "agentName": context.metadata.agent_name,
                    "agentPhase": "Errored",
                    "executionID": run.metadata.name,
                    "executionPhase": event["object"]["status"]["phase"],
                    "error": "Argo workflow execution failed"
                })
                raise ArgoWorkflowException(context.status["workflow-name"], event)

            context.status["event"] = event
            self.notify_observers("ArgoExecutor", {
                    "agentName": context.metadata.agent_name,
                    "agentPhase": "Running",
                    "executionID": run.metadata.name,
                    "executionPhase": event["object"]["status"]["phase"],
                    "error": None
            })

        # Return execution context
        return context

    def get_results(self, context: ExecutionContext,timeout_ms: int = 60000) -> dict:
        if not "event" in context.status:
            ret, event = self._wait_for_crd_status(context.status["workflow-name"],context.spec["program"],timeout_ms)
            if not ret:
                self.notify_observers("ArgoExecutor", {
                    "agentName": context.metadata.agent_name,
                    "agentPhase": "Errored",
                    "executionID": event["object"]["metadata"]["name"],
                    "executionPhase": event["object"]["status"]["phase"],
                    "error": "Argo workflow execution failed"
                })
                raise ArgoWorkflowException(context.status["workflow-name"], event)

            self.notify_observers("ArgoExecutor", {
                    "agentName": context.metadata.agent_name,
                    "agentPhase": "Running",
                    "executionID": event["object"]["metadata"]["name"],
                    "executionPhase": event["object"]["status"]["phase"],
                    "error": None
            })
            context.status["event"] = event

        # Return the output
        return self._extract_output(context.status["event"])

    def _extract_output(self, workflow_event: dict) -> dict:
        # Get the execution nodes
        nodes = workflow_event["object"]["status"]["nodes"]

        # Look for the node that run the 'agent-runner' template
        output = None
        for node in nodes.values():
            try:
                if node["templateName"] == "agent-runner":
                    # Get the output of the job
                    output = node["outputs"]["parameters"][0]["value"]
                    break
            except KeyError:
                pass

        if not output:
            raise Exception("Output not found in workflow")

        return json.loads(output)

    def _get_namespace(self):
        if DEBUG:
            return "argo"

        with open(
            "/var/run/secrets/kubernetes.io/serviceaccount/namespace", "r"
        ) as file:
            return file.read().strip()

    def _wait_for_crd_status(self, name, program, timeout_ms=60000):
        start_time = time.time()
        w = watch.Watch()

        print(f"Waiting for CRD status to change to Succeeded - {name} - {program}")
        crd_api = client.CustomObjectsApi()

        for event in w.stream(
            crd_api.list_namespaced_custom_object,
            group="argoproj.io",
            version="v1alpha1",
            plural="workflows",
            namespace=self.namespace,
            field_selector=f"metadata.name={name}",
            watch=True,
            timeout_seconds=int(timeout_ms / 1000),
        ):
            try:
                if event["object"]["status"]["finishedAt"] is not None:
                    if (
                        event["object"]["status"]["phase"] == "Failed"
                        or event["object"]["status"]["phase"] == "Error"
                    ):
                        print(f"Workflow failed - {name}")
                        return False, event

                    print(f"Workflow completed - {name}")
                    return True, event
            except KeyError:
                print("KeyError - Status not found in CRD")
                pass

        # Timeout
        raise Exception("Timeout waiting for CRD status")

    def _get_workflow_owner(self, workflow) -> client.V1OwnerReference | None:
        pod_name = os.getenv("POD_NAME", None)

        if not pod_name:
            return None

        v1_api = client.CoreV1Api()

        # Retrieve the Pod object (note: this pod is the current pod running this script)
        pod = v1_api.read_namespaced_pod(name=pod_name, namespace=self.namespace)

        # Set the owner reference to the workflow
        owner_reference = client.V1OwnerReference(
            api_version=pod.api_version,
            kind=pod.kind,
            name=pod.metadata.name,
            uid=pod.metadata.uid,
            controller=True,
            block_owner_deletion=True,
        )

        return owner_reference
