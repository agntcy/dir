from google.protobuf import struct_pb2 as _struct_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class LLMInfo(_message.Message):
    __slots__ = ("provider", "version", "model_name", "annotations", "config", "endpoint", "capabilities")
    class AnnotationsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    PROVIDER_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    MODEL_NAME_FIELD_NUMBER: _ClassVar[int]
    ANNOTATIONS_FIELD_NUMBER: _ClassVar[int]
    CONFIG_FIELD_NUMBER: _ClassVar[int]
    ENDPOINT_FIELD_NUMBER: _ClassVar[int]
    CAPABILITIES_FIELD_NUMBER: _ClassVar[int]
    provider: str
    version: str
    model_name: str
    annotations: _containers.ScalarMap[str, str]
    config: LLMConfig
    endpoint: LLMEndpoint
    capabilities: LLMCapabilities
    def __init__(self, provider: _Optional[str] = ..., version: _Optional[str] = ..., model_name: _Optional[str] = ..., annotations: _Optional[_Mapping[str, str]] = ..., config: _Optional[_Union[LLMConfig, _Mapping]] = ..., endpoint: _Optional[_Union[LLMEndpoint, _Mapping]] = ..., capabilities: _Optional[_Union[LLMCapabilities, _Mapping]] = ...) -> None: ...

class LLMConfig(_message.Message):
    __slots__ = ("temperature", "max_tokens", "stop_sequences", "top_p", "frequency_penalty", "presence_penalty", "response_format", "seed", "provider_config")
    TEMPERATURE_FIELD_NUMBER: _ClassVar[int]
    MAX_TOKENS_FIELD_NUMBER: _ClassVar[int]
    STOP_SEQUENCES_FIELD_NUMBER: _ClassVar[int]
    TOP_P_FIELD_NUMBER: _ClassVar[int]
    FREQUENCY_PENALTY_FIELD_NUMBER: _ClassVar[int]
    PRESENCE_PENALTY_FIELD_NUMBER: _ClassVar[int]
    RESPONSE_FORMAT_FIELD_NUMBER: _ClassVar[int]
    SEED_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_CONFIG_FIELD_NUMBER: _ClassVar[int]
    temperature: float
    max_tokens: int
    stop_sequences: _containers.RepeatedScalarFieldContainer[str]
    top_p: float
    frequency_penalty: float
    presence_penalty: float
    response_format: str
    seed: int
    provider_config: _struct_pb2.Struct
    def __init__(self, temperature: _Optional[float] = ..., max_tokens: _Optional[int] = ..., stop_sequences: _Optional[_Iterable[str]] = ..., top_p: _Optional[float] = ..., frequency_penalty: _Optional[float] = ..., presence_penalty: _Optional[float] = ..., response_format: _Optional[str] = ..., seed: _Optional[int] = ..., provider_config: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ...) -> None: ...

class LLMEndpoint(_message.Message):
    __slots__ = ("base_url", "api_version", "headers")
    class HeadersEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    BASE_URL_FIELD_NUMBER: _ClassVar[int]
    API_VERSION_FIELD_NUMBER: _ClassVar[int]
    HEADERS_FIELD_NUMBER: _ClassVar[int]
    base_url: str
    api_version: str
    headers: _containers.ScalarMap[str, str]
    def __init__(self, base_url: _Optional[str] = ..., api_version: _Optional[str] = ..., headers: _Optional[_Mapping[str, str]] = ...) -> None: ...

class LLMCapabilities(_message.Message):
    __slots__ = ("max_context_tokens", "max_response_tokens", "features", "supported_formats", "specializations", "cost_per_1k_prompt_tokens", "cost_per_1k_completion_tokens")
    MAX_CONTEXT_TOKENS_FIELD_NUMBER: _ClassVar[int]
    MAX_RESPONSE_TOKENS_FIELD_NUMBER: _ClassVar[int]
    FEATURES_FIELD_NUMBER: _ClassVar[int]
    SUPPORTED_FORMATS_FIELD_NUMBER: _ClassVar[int]
    SPECIALIZATIONS_FIELD_NUMBER: _ClassVar[int]
    COST_PER_1K_PROMPT_TOKENS_FIELD_NUMBER: _ClassVar[int]
    COST_PER_1K_COMPLETION_TOKENS_FIELD_NUMBER: _ClassVar[int]
    max_context_tokens: int
    max_response_tokens: int
    features: _containers.RepeatedScalarFieldContainer[str]
    supported_formats: _containers.RepeatedScalarFieldContainer[str]
    specializations: _containers.RepeatedScalarFieldContainer[str]
    cost_per_1k_prompt_tokens: float
    cost_per_1k_completion_tokens: float
    def __init__(self, max_context_tokens: _Optional[int] = ..., max_response_tokens: _Optional[int] = ..., features: _Optional[_Iterable[str]] = ..., supported_formats: _Optional[_Iterable[str]] = ..., specializations: _Optional[_Iterable[str]] = ..., cost_per_1k_prompt_tokens: _Optional[float] = ..., cost_per_1k_completion_tokens: _Optional[float] = ...) -> None: ...

class LLMQuota(_message.Message):
    __slots__ = ("requests_per_minute", "tokens_per_minute", "max_concurrent_requests", "quota_reset_period", "enable_quota_tracking")
    REQUESTS_PER_MINUTE_FIELD_NUMBER: _ClassVar[int]
    TOKENS_PER_MINUTE_FIELD_NUMBER: _ClassVar[int]
    MAX_CONCURRENT_REQUESTS_FIELD_NUMBER: _ClassVar[int]
    QUOTA_RESET_PERIOD_FIELD_NUMBER: _ClassVar[int]
    ENABLE_QUOTA_TRACKING_FIELD_NUMBER: _ClassVar[int]
    requests_per_minute: int
    tokens_per_minute: int
    max_concurrent_requests: int
    quota_reset_period: int
    enable_quota_tracking: bool
    def __init__(self, requests_per_minute: _Optional[int] = ..., tokens_per_minute: _Optional[int] = ..., max_concurrent_requests: _Optional[int] = ..., quota_reset_period: _Optional[int] = ..., enable_quota_tracking: bool = ...) -> None: ...
