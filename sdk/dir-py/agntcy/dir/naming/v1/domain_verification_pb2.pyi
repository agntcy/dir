from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Mapping as _Mapping, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class DomainVerification(_message.Message):
    __slots__ = ("domain", "method", "matched_key_id", "verified_at")
    DOMAIN_FIELD_NUMBER: _ClassVar[int]
    METHOD_FIELD_NUMBER: _ClassVar[int]
    MATCHED_KEY_ID_FIELD_NUMBER: _ClassVar[int]
    VERIFIED_AT_FIELD_NUMBER: _ClassVar[int]
    domain: str
    method: str
    matched_key_id: str
    verified_at: _timestamp_pb2.Timestamp
    def __init__(self, domain: _Optional[str] = ..., method: _Optional[str] = ..., matched_key_id: _Optional[str] = ..., verified_at: _Optional[_Union[_timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...
