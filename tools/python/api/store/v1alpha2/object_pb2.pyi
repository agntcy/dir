from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ObjectType(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    OBJECT_TYPE_UNSPECIFIED: _ClassVar[ObjectType]
    OBJECT_TYPE_RAW: _ClassVar[ObjectType]
OBJECT_TYPE_UNSPECIFIED: ObjectType
OBJECT_TYPE_RAW: ObjectType

class ObjectRef(_message.Message):
    __slots__ = ("cid",)
    CID_FIELD_NUMBER: _ClassVar[int]
    cid: str
    def __init__(self, cid: _Optional[str] = ...) -> None: ...

class Object(_message.Message):
    __slots__ = ("cid", "type", "annotations", "created_at", "size", "data")
    class AnnotationsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    CID_FIELD_NUMBER: _ClassVar[int]
    TYPE_FIELD_NUMBER: _ClassVar[int]
    ANNOTATIONS_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    SIZE_FIELD_NUMBER: _ClassVar[int]
    DATA_FIELD_NUMBER: _ClassVar[int]
    cid: str
    type: ObjectType
    annotations: _containers.ScalarMap[str, str]
    created_at: str
    size: int
    data: bytes
    def __init__(self, cid: _Optional[str] = ..., type: _Optional[_Union[ObjectType, str]] = ..., annotations: _Optional[_Mapping[str, str]] = ..., created_at: _Optional[str] = ..., size: _Optional[int] = ..., data: _Optional[bytes] = ...) -> None: ...
