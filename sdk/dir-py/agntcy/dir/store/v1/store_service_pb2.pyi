from agntcy.dir.core.v1 import cid_pb2 as _cid_pb2
from agntcy.dir.core.v1 import record_pb2 as _record_pb2
from google.protobuf import empty_pb2 as _empty_pb2
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
    __slots__ = ("success", "error_message", "cid")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    ERROR_MESSAGE_FIELD_NUMBER: _ClassVar[int]
    CID_FIELD_NUMBER: _ClassVar[int]
    success: bool
    error_message: str
    cid: _cid_pb2.CID
    def __init__(self, success: bool = ..., error_message: _Optional[str] = ..., cid: _Optional[_Union[_cid_pb2.CID, _Mapping]] = ...) -> None: ...

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
