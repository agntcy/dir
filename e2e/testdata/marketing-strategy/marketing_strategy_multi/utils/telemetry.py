# SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
# SPDX-License-Identifier: Apache-2.0

from os import environ

from opentelemetry.sdk.trace import TracerProvider

from openinference.instrumentation.crewai import CrewAIInstrumentor
from openinference.instrumentation.langchain import LangChainInstrumentor
from opentelemetry.exporter.otlp.proto.http.trace_exporter import OTLPSpanExporter
from opentelemetry.sdk.trace.export import SimpleSpanProcessor
from opentelemetry.sdk.resources import SERVICE_NAME, Resource


def instrument(app_name: str) -> bool:
    ret = False

    telemetry_endpoint = environ.get("TELEMETRY_ENDPOINT")
    if telemetry_endpoint is not None:
        print("Enable telemtry, endpoint = " + telemetry_endpoint)
        endpoint = telemetry_endpoint + "/v1/traces"
        resource = Resource(
            attributes={SERVICE_NAME: app_name},
        )
        trace_provider = TracerProvider(resource=resource)
        trace_provider.add_span_processor(
            SimpleSpanProcessor(OTLPSpanExporter(endpoint))
        )
        CrewAIInstrumentor().instrument(tracer_provider=trace_provider)
        LangChainInstrumentor().instrument(tracer_provider=trace_provider)
        ret = True
    else:
        print("Telemetry endpoint not set, skipping telemetry initialization")

    return ret
