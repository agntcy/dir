from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional

DESCRIPTOR: _descriptor.FileDescriptor

class Object(_message.Message):
    __slots__ = ("cid", "type", "size", "annotations", "data")
    class AnnotationsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    CID_FIELD_NUMBER: _ClassVar[int]
    TYPE_FIELD_NUMBER: _ClassVar[int]
    SIZE_FIELD_NUMBER: _ClassVar[int]
    ANNOTATIONS_FIELD_NUMBER: _ClassVar[int]
    DATA_FIELD_NUMBER: _ClassVar[int]
    cid: str
    type: str
    size: int
    annotations: _containers.ScalarMap[str, str]
    data: bytes
    def __init__(self, cid: _Optional[str] = ..., type: _Optional[str] = ..., size: _Optional[int] = ..., annotations: _Optional[_Mapping[str, str]] = ..., data: _Optional[bytes] = ...) -> None: ...

class ObjectRef(_message.Message):
    __slots__ = ("cid",)
    CID_FIELD_NUMBER: _ClassVar[int]
    cid: str
    def __init__(self, cid: _Optional[str] = ...) -> None: ...
