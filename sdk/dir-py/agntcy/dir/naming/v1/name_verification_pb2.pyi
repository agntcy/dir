from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Mapping as _Mapping, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Verification(_message.Message):
    __slots__ = ("domain",)
    DOMAIN_FIELD_NUMBER: _ClassVar[int]
    domain: DomainVerification
    def __init__(self, domain: _Optional[_Union[DomainVerification, _Mapping]] = ...) -> None: ...

class DomainVerification(_message.Message):
    __slots__ = ("domain", "method", "key_id", "verified_at")
    DOMAIN_FIELD_NUMBER: _ClassVar[int]
    METHOD_FIELD_NUMBER: _ClassVar[int]
    KEY_ID_FIELD_NUMBER: _ClassVar[int]
    VERIFIED_AT_FIELD_NUMBER: _ClassVar[int]
    domain: str
    method: str
    key_id: str
    verified_at: _timestamp_pb2.Timestamp
    def __init__(self, domain: _Optional[str] = ..., method: _Optional[str] = ..., key_id: _Optional[str] = ..., verified_at: _Optional[_Union[_timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...
