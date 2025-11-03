from google.protobuf import struct_pb2 as _struct_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Mapping as _Mapping, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class RecordRef(_message.Message):
    __slots__ = ("cid",)
    CID_FIELD_NUMBER: _ClassVar[int]
    cid: str
    def __init__(self, cid: _Optional[str] = ...) -> None: ...

class RecordMeta(_message.Message):
    __slots__ = ("cid", "annotations", "schema_version", "created_at")
    class AnnotationsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    CID_FIELD_NUMBER: _ClassVar[int]
    ANNOTATIONS_FIELD_NUMBER: _ClassVar[int]
    SCHEMA_VERSION_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    cid: str
    annotations: _containers.ScalarMap[str, str]
    schema_version: str
    created_at: str
    def __init__(self, cid: _Optional[str] = ..., annotations: _Optional[_Mapping[str, str]] = ..., schema_version: _Optional[str] = ..., created_at: _Optional[str] = ...) -> None: ...

class Record(_message.Message):
    __slots__ = ("data",)
    DATA_FIELD_NUMBER: _ClassVar[int]
    data: _struct_pb2.Struct
    def __init__(self, data: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ...) -> None: ...

class RecordReferrer(_message.Message):
    __slots__ = ("type", "record_ref", "annotations", "created_at", "data")
    class AnnotationsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    TYPE_FIELD_NUMBER: _ClassVar[int]
    RECORD_REF_FIELD_NUMBER: _ClassVar[int]
    ANNOTATIONS_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    DATA_FIELD_NUMBER: _ClassVar[int]
    type: str
    record_ref: RecordRef
    annotations: _containers.ScalarMap[str, str]
    created_at: str
    data: _struct_pb2.Struct
    def __init__(self, type: _Optional[str] = ..., record_ref: _Optional[_Union[RecordRef, _Mapping]] = ..., annotations: _Optional[_Mapping[str, str]] = ..., created_at: _Optional[str] = ..., data: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ...) -> None: ...
