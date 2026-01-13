from agntcy.dir.naming.v1 import domain_verification_pb2 as _domain_verification_pb2
from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Iterable as _Iterable, Mapping as _Mapping, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class VerifyDomainRequest(_message.Message):
    __slots__ = ("cid",)
    CID_FIELD_NUMBER: _ClassVar[int]
    cid: str
    def __init__(self, cid: _Optional[str] = ...) -> None: ...

class VerifyDomainResponse(_message.Message):
    __slots__ = ("verified", "verification", "error_message")
    VERIFIED_FIELD_NUMBER: _ClassVar[int]
    VERIFICATION_FIELD_NUMBER: _ClassVar[int]
    ERROR_MESSAGE_FIELD_NUMBER: _ClassVar[int]
    verified: bool
    verification: _domain_verification_pb2.DomainVerification
    error_message: str
    def __init__(self, verified: bool = ..., verification: _Optional[_Union[_domain_verification_pb2.DomainVerification, _Mapping]] = ..., error_message: _Optional[str] = ...) -> None: ...

class CheckDomainVerificationRequest(_message.Message):
    __slots__ = ("cid",)
    CID_FIELD_NUMBER: _ClassVar[int]
    cid: str
    def __init__(self, cid: _Optional[str] = ...) -> None: ...

class CheckDomainVerificationResponse(_message.Message):
    __slots__ = ("verified", "verification", "error_message")
    VERIFIED_FIELD_NUMBER: _ClassVar[int]
    VERIFICATION_FIELD_NUMBER: _ClassVar[int]
    ERROR_MESSAGE_FIELD_NUMBER: _ClassVar[int]
    verified: bool
    verification: _domain_verification_pb2.DomainVerification
    error_message: str
    def __init__(self, verified: bool = ..., verification: _Optional[_Union[_domain_verification_pb2.DomainVerification, _Mapping]] = ..., error_message: _Optional[str] = ...) -> None: ...

class ListVerifiedAgentsRequest(_message.Message):
    __slots__ = ("domain", "limit", "page_token")
    DOMAIN_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    PAGE_TOKEN_FIELD_NUMBER: _ClassVar[int]
    domain: str
    limit: int
    page_token: str
    def __init__(self, domain: _Optional[str] = ..., limit: _Optional[int] = ..., page_token: _Optional[str] = ...) -> None: ...

class ListVerifiedAgentsResponse(_message.Message):
    __slots__ = ("agents", "next_page_token")
    AGENTS_FIELD_NUMBER: _ClassVar[int]
    NEXT_PAGE_TOKEN_FIELD_NUMBER: _ClassVar[int]
    agents: _containers.RepeatedCompositeFieldContainer[VerifiedAgent]
    next_page_token: str
    def __init__(self, agents: _Optional[_Iterable[_Union[VerifiedAgent, _Mapping]]] = ..., next_page_token: _Optional[str] = ...) -> None: ...

class VerifiedAgent(_message.Message):
    __slots__ = ("cid", "name", "version", "verified_at", "method")
    CID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    VERIFIED_AT_FIELD_NUMBER: _ClassVar[int]
    METHOD_FIELD_NUMBER: _ClassVar[int]
    cid: str
    name: str
    version: str
    verified_at: _timestamp_pb2.Timestamp
    method: str
    def __init__(self, cid: _Optional[str] = ..., name: _Optional[str] = ..., version: _Optional[str] = ..., verified_at: _Optional[_Union[_timestamp_pb2.Timestamp, _Mapping]] = ..., method: _Optional[str] = ...) -> None: ...
