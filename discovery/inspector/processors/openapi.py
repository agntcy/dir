"""
OpenAPI specification processor.

Discovers and extracts OpenAPI/Swagger specifications from workloads.
"""

import json
from dataclasses import dataclass, field, asdict
from datetime import datetime
from typing import Optional

import requests

from config import ProcessorConfig, logger
from models import Workload
from processors.interface import ProcessorInterface


# ==================== Result Model ====================

@dataclass
class OpenAPIResult:
    """OpenAPI discovery result."""
    found: bool
    endpoint: Optional[str] = None
    version: Optional[str] = None
    title: Optional[str] = None
    paths_count: Optional[int] = None
    spec: Optional[dict] = None
    error: Optional[str] = None
    discovered_at: str = field(default_factory=lambda: datetime.utcnow().isoformat())
    
    def to_dict(self) -> dict:
        # Don't include full spec in serialization by default
        d = asdict(self)
        if d.get("spec"):
            d["has_spec"] = True
            del d["spec"]
        return d
    
    def to_json(self) -> str:
        return json.dumps(self.to_dict())


# ==================== Processor ====================

class OpenAPIProcessor(ProcessorInterface):
    """
    OpenAPI specification processor.
    
    Discovers OpenAPI/Swagger specs from workloads and extracts:
    - Spec location
    - API version and title
    - Number of endpoints
    - Optionally stores the full spec
    """
    
    def __init__(self, config: ProcessorConfig):
        self.config = config
        self.timeout = config.openapi_timeout
        self.paths = config.openapi_paths
        self._enabled = config.openapi_enabled
    
    @property
    def name(self) -> str:
        return "openapi"
    
    @property
    def enabled(self) -> bool:
        return self._enabled
    
    def should_process(self, workload: Workload) -> bool:
        """Only process workloads with addresses and ports."""
        # Could also check for labels like 'openapi=true' or 'api=true'
        return bool(workload.addresses) and bool(workload.ports)
    
    def process(self, workload: Workload) -> Optional[dict]:
        """
        Discover OpenAPI specification from workload.
        
        Tries common OpenAPI/Swagger endpoints and parses the spec.
        """
        if not self.should_process(workload):
            return OpenAPIResult(
                found=False,
                error="No addresses or ports available"
            ).to_dict()
        
        # Build list of URLs to try
        urls_to_try = []
        for addr in workload.addresses:
            for port in workload.ports:
                for path in self.paths:
                    url = f"http://{addr}:{port}{path}"
                    urls_to_try.append(url)
        
        # Try each URL
        for url in urls_to_try:
            result = self._fetch_spec(url)
            if result.found:
                logger.info(
                    "[openapi] %s has spec at %s (%s v%s, %d paths)",
                    workload.name,
                    result.endpoint,
                    result.title or "Untitled",
                    result.version or "?",
                    result.paths_count or 0
                )
                return result.to_dict()
        
        # No spec found
        logger.debug("[openapi] %s has no OpenAPI spec", workload.name)
        return OpenAPIResult(
            found=False,
            error=f"No OpenAPI spec found (tried {len(urls_to_try)} URLs)"
        ).to_dict()
    
    def _fetch_spec(self, url: str) -> OpenAPIResult:
        """Fetch and parse OpenAPI spec from URL."""
        try:
            response = requests.get(
                url,
                timeout=self.timeout,
                headers={"Accept": "application/json"}
            )
            
            if response.status_code != 200:
                return OpenAPIResult(found=False, error=f"HTTP {response.status_code}")
            
            # Try to parse as JSON
            try:
                spec = response.json()
            except Exception:
                return OpenAPIResult(found=False, error="Invalid JSON")
            
            # Validate it looks like an OpenAPI spec
            if not self._is_openapi_spec(spec):
                return OpenAPIResult(found=False, error="Not an OpenAPI spec")
            
            # Extract metadata
            return OpenAPIResult(
                found=True,
                endpoint=url,
                version=self._get_version(spec),
                title=self._get_title(spec),
                paths_count=self._count_paths(spec),
                spec=spec  # Store full spec
            )
        
        except requests.exceptions.Timeout:
            return OpenAPIResult(found=False, error="Timeout")
        except requests.exceptions.ConnectionError:
            return OpenAPIResult(found=False, error="Connection error")
        except Exception as e:
            return OpenAPIResult(found=False, error=str(e)[:100])
    
    def _is_openapi_spec(self, spec: dict) -> bool:
        """Check if dict looks like an OpenAPI/Swagger spec."""
        # OpenAPI 3.x
        if "openapi" in spec and "paths" in spec:
            return True
        # Swagger 2.x
        if "swagger" in spec and "paths" in spec:
            return True
        # OpenAPI 3.1 can have webhooks instead of paths
        if "openapi" in spec and "webhooks" in spec:
            return True
        return False
    
    def _get_version(self, spec: dict) -> Optional[str]:
        """Extract API version from spec."""
        # OpenAPI 3.x
        if "openapi" in spec:
            return spec.get("openapi")
        # Swagger 2.x
        if "swagger" in spec:
            return spec.get("swagger")
        return None
    
    def _get_title(self, spec: dict) -> Optional[str]:
        """Extract API title from spec."""
        info = spec.get("info", {})
        return info.get("title")
    
    def _count_paths(self, spec: dict) -> int:
        """Count number of paths/endpoints in spec."""
        paths = spec.get("paths", {})
        return len(paths)
