from google.protobuf import struct_pb2 as _struct_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Iterable as _Iterable, Mapping as _Mapping, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class RecordRef(_message.Message):
    __slots__ = ("cid",)
    CID_FIELD_NUMBER: _ClassVar[int]
    cid: str
    def __init__(self, cid: _Optional[str] = ...) -> None: ...

class RecordMeta(_message.Message):
    __slots__ = ("cid", "annotations", "created_at", "type", "links", "parent", "size")
    class AnnotationsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    CID_FIELD_NUMBER: _ClassVar[int]
    ANNOTATIONS_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    TYPE_FIELD_NUMBER: _ClassVar[int]
    LINKS_FIELD_NUMBER: _ClassVar[int]
    PARENT_FIELD_NUMBER: _ClassVar[int]
    SIZE_FIELD_NUMBER: _ClassVar[int]
    cid: str
    annotations: _containers.ScalarMap[str, str]
    created_at: str
    type: str
    links: _containers.RepeatedCompositeFieldContainer[RecordRef]
    parent: RecordRef
    size: int
    def __init__(self, cid: _Optional[str] = ..., annotations: _Optional[_Mapping[str, str]] = ..., created_at: _Optional[str] = ..., type: _Optional[str] = ..., links: _Optional[_Iterable[_Union[RecordRef, _Mapping]]] = ..., parent: _Optional[_Union[RecordRef, _Mapping]] = ..., size: _Optional[int] = ...) -> None: ...

class Record(_message.Message):
    __slots__ = ("meta", "data")
    META_FIELD_NUMBER: _ClassVar[int]
    DATA_FIELD_NUMBER: _ClassVar[int]
    meta: RecordMeta
    data: _struct_pb2.Struct
    def __init__(self, meta: _Optional[_Union[RecordMeta, _Mapping]] = ..., data: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ...) -> None: ...
