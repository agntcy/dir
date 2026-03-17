from agntcy.dir.core.v1 import record_pb2 as _record_pb2
from agntcy.dir.core.v1 import referrer_pb2 as _referrer_pb2
from google.protobuf import empty_pb2 as _empty_pb2
from google.rpc import status_pb2 as _status_pb2
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Mapping as _Mapping, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class PushReferrerRequest(_message.Message):
    __slots__ = ("record_ref", "referrer")
    RECORD_REF_FIELD_NUMBER: _ClassVar[int]
    REFERRER_FIELD_NUMBER: _ClassVar[int]
    record_ref: _record_pb2.RecordRef
    referrer: _record_pb2.RecordReferrer
    def __init__(self, record_ref: _Optional[_Union[_record_pb2.RecordRef, _Mapping]] = ..., referrer: _Optional[_Union[_record_pb2.RecordReferrer, _Mapping]] = ...) -> None: ...

class PushReferrerResponse(_message.Message):
    __slots__ = ("referrer_ref", "error")
    REFERRER_REF_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    referrer_ref: _referrer_pb2.ReferrerRef
    error: _status_pb2.Status
    def __init__(self, referrer_ref: _Optional[_Union[_referrer_pb2.ReferrerRef, _Mapping]] = ..., error: _Optional[_Union[_status_pb2.Status, _Mapping]] = ...) -> None: ...

class PullReferrerRequest(_message.Message):
    __slots__ = ("record_ref", "referrer_type")
    RECORD_REF_FIELD_NUMBER: _ClassVar[int]
    REFERRER_TYPE_FIELD_NUMBER: _ClassVar[int]
    record_ref: _record_pb2.RecordRef
    referrer_type: str
    def __init__(self, record_ref: _Optional[_Union[_record_pb2.RecordRef, _Mapping]] = ..., referrer_type: _Optional[str] = ...) -> None: ...

class PullReferrerResponse(_message.Message):
    __slots__ = ("referrer",)
    REFERRER_FIELD_NUMBER: _ClassVar[int]
    referrer: _record_pb2.RecordReferrer
    def __init__(self, referrer: _Optional[_Union[_record_pb2.RecordReferrer, _Mapping]] = ...) -> None: ...
