from agntcy.dir.core.v1 import record_pb2 as _record_pb2
from agntcy.dir.core.v1 import referrer_pb2 as _referrer_pb2
from google.protobuf import empty_pb2 as _empty_pb2
from google.protobuf import struct_pb2 as _struct_pb2
from buf.validate import validate_pb2 as _validate_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Mapping as _Mapping, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class PushReferrerRequest(_message.Message):
    __slots__ = ("record_ref", "type", "annotations", "created_at", "data")
    class AnnotationsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    RECORD_REF_FIELD_NUMBER: _ClassVar[int]
    TYPE_FIELD_NUMBER: _ClassVar[int]
    ANNOTATIONS_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    DATA_FIELD_NUMBER: _ClassVar[int]
    record_ref: _record_pb2.RecordRef
    type: str
    annotations: _containers.ScalarMap[str, str]
    created_at: str
    data: _struct_pb2.Struct
    def __init__(self, record_ref: _Optional[_Union[_record_pb2.RecordRef, _Mapping]] = ..., type: _Optional[str] = ..., annotations: _Optional[_Mapping[str, str]] = ..., created_at: _Optional[str] = ..., data: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ...) -> None: ...

class PushReferrerResponse(_message.Message):
    __slots__ = ("success", "error_message", "referrer_ref")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    ERROR_MESSAGE_FIELD_NUMBER: _ClassVar[int]
    REFERRER_REF_FIELD_NUMBER: _ClassVar[int]
    success: bool
    error_message: str
    referrer_ref: _referrer_pb2.ReferrerRef
    def __init__(self, success: bool = ..., error_message: _Optional[str] = ..., referrer_ref: _Optional[_Union[_referrer_pb2.ReferrerRef, _Mapping]] = ...) -> None: ...

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
