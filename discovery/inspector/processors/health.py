"""
Health check processor.

Probes workloads for health endpoints and records results.
"""

import json
import time
from dataclasses import dataclass, field, asdict
from datetime import datetime
from typing import Optional

import requests

from config import ProcessorConfig, logger
from models import Workload
from processors.interface import ProcessorInterface


# ==================== Result Model ====================

@dataclass
class HealthResult:
    """Health check result."""
    healthy: bool
    endpoint: Optional[str] = None
    status_code: Optional[int] = None
    response_time_ms: Optional[float] = None
    error: Optional[str] = None
    checked_at: str = field(default_factory=lambda: datetime.utcnow().isoformat())
    
    def to_dict(self) -> dict:
        return asdict(self)
    
    def to_json(self) -> str:
        return json.dumps(self.to_dict())


# ==================== Processor ====================

class HealthProcessor(ProcessorInterface):
    """
    Health check processor.
    
    Probes common health endpoints on workloads and records:
    - Whether the workload is healthy
    - Which endpoint responded
    - Response time
    """
    
    def __init__(self, config: ProcessorConfig):
        self.config = config
        self.timeout = config.health_timeout
        self.paths = config.health_paths
        self._enabled = config.health_enabled
    
    @property
    def name(self) -> str:
        return "health"
    
    @property
    def enabled(self) -> bool:
        return self._enabled
    
    def should_process(self, workload: Workload) -> bool:
        """Only process workloads with addresses and ports."""
        return bool(workload.addresses) and bool(workload.ports)
    
    def process(self, workload: Workload) -> Optional[dict]:
        """
        Probe health endpoints on the workload.
        
        Tries each address:port combination with each health path.
        Returns on first successful response.
        """
        if not self.should_process(workload):
            return HealthResult(
                healthy=False,
                error="No addresses or ports available"
            ).to_dict()
        
        # Build list of URLs to try
        urls_to_try = []
        for addr in workload.addresses:
            for port in workload.ports:
                for path in self.paths:
                    urls_to_try.append(f"http://{addr}:{port}{path}")
        
        logger.info("[health] Probing (%s) URLs for workload %s", ','.join(urls_to_try), workload.name)

        # Try each URL
        for url in urls_to_try:
            result = self._probe_url(url)
            if result.healthy:
                logger.info(
                    "[health] %s is healthy at %s (%.0fms)",
                    workload.name,
                    result.endpoint,
                    result.response_time_ms or 0
                )
                return result.to_dict()
            else:
                logger.warning(
                    "[health] %s health probe failed at %s: %s",
                    workload.name,
                    result.endpoint,
                    result.error or f"HTTP {result.status_code}"
                )
        
        # All probes failed
        logger.warning("[health] %s is unhealthy - no endpoints responded", workload.name)
        return HealthResult(
            healthy=False,
            error=f"No health endpoints responded (tried {len(urls_to_try)} URLs)"
        ).to_dict()
    
    def _probe_url(self, url: str) -> HealthResult:
        """Probe a single URL."""
        try:
            start = time.time()
            response = requests.get(url, timeout=self.timeout, allow_redirects=True)
            elapsed_ms = (time.time() - start) * 1000
            
            # Consider 2xx and some 3xx as healthy
            if response.status_code < 400:
                return HealthResult(
                    healthy=True,
                    endpoint=url,
                    status_code=response.status_code,
                    response_time_ms=round(elapsed_ms, 2)
                )
            else:
                return HealthResult(
                    healthy=False,
                    endpoint=url,
                    status_code=response.status_code,
                    response_time_ms=round(elapsed_ms, 2),
                    error=f"HTTP {response.status_code}"
                )
        
        except requests.exceptions.Timeout:
            return HealthResult(
                healthy=False,
                endpoint=url,
                error="Timeout"
            )
        except requests.exceptions.ConnectionError as e:
            return HealthResult(
                healthy=False,
                endpoint=url,
                error=f"Connection error: {str(e)[:100]}"
            )
        except Exception as e:
            return HealthResult(
                healthy=False,
                endpoint=url,
                error=f"Error: {str(e)[:100]}"
            )
