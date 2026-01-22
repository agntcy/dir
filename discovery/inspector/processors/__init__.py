"""
Processors module for inspector.
"""

from typing import List

from processors.interface import ProcessorInterface
from processors.health import HealthProcessor
from processors.openapi import OpenAPIProcessor
from config import logger, Config


def create_processors(config: Config) -> List[ProcessorInterface]:
    """Initialize enabled processors."""
    processors: List[ProcessorInterface] = []

    # Health check processor
    health = HealthProcessor(config.processor)
    if health.enabled:
        processors.append(health)
        logger.info("Enabled processor: %s", health.name)
    
    # OpenAPI processor
    openapi = OpenAPIProcessor(config.processor)
    if openapi.enabled:
        processors.append(openapi)
        logger.info("Enabled processor: %s", openapi.name)
    
    return processors
