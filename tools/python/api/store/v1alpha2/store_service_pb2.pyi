from core.v1 import object_pb2 as _object_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class PushRequestChunk(_message.Message):
    __slots__ = ("object_ref", "object_type", "annotations", "size", "data")
    class AnnotationsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    OBJECT_REF_FIELD_NUMBER: _ClassVar[int]
    OBJECT_TYPE_FIELD_NUMBER: _ClassVar[int]
    ANNOTATIONS_FIELD_NUMBER: _ClassVar[int]
    SIZE_FIELD_NUMBER: _ClassVar[int]
    DATA_FIELD_NUMBER: _ClassVar[int]
    object_ref: _object_pb2.ObjectRef
    object_type: _object_pb2.ObjectType
    annotations: _containers.ScalarMap[str, str]
    size: int
    data: bytes
    def __init__(self, object_ref: _Optional[_Union[_object_pb2.ObjectRef, _Mapping]] = ..., object_type: _Optional[_Union[_object_pb2.ObjectType, str]] = ..., annotations: _Optional[_Mapping[str, str]] = ..., size: _Optional[int] = ..., data: _Optional[bytes] = ...) -> None: ...

class PushResponse(_message.Message):
    __slots__ = ("object_ref",)
    OBJECT_REF_FIELD_NUMBER: _ClassVar[int]
    object_ref: _object_pb2.ObjectRef
    def __init__(self, object_ref: _Optional[_Union[_object_pb2.ObjectRef, _Mapping]] = ...) -> None: ...

class PullRequest(_message.Message):
    __slots__ = ("object_ref",)
    OBJECT_REF_FIELD_NUMBER: _ClassVar[int]
    object_ref: _object_pb2.ObjectRef
    def __init__(self, object_ref: _Optional[_Union[_object_pb2.ObjectRef, _Mapping]] = ...) -> None: ...

class PullResponseChunk(_message.Message):
    __slots__ = ("data", "size")
    DATA_FIELD_NUMBER: _ClassVar[int]
    SIZE_FIELD_NUMBER: _ClassVar[int]
    data: bytes
    size: int
    def __init__(self, data: _Optional[bytes] = ..., size: _Optional[int] = ...) -> None: ...

class LookupRequest(_message.Message):
    __slots__ = ("object_ref",)
    OBJECT_REF_FIELD_NUMBER: _ClassVar[int]
    object_ref: _object_pb2.ObjectRef
    def __init__(self, object_ref: _Optional[_Union[_object_pb2.ObjectRef, _Mapping]] = ...) -> None: ...

class LookupResponse(_message.Message):
    __slots__ = ("object",)
    OBJECT_FIELD_NUMBER: _ClassVar[int]
    object: _object_pb2.Object
    def __init__(self, object: _Optional[_Union[_object_pb2.Object, _Mapping]] = ...) -> None: ...

class DeleteRequest(_message.Message):
    __slots__ = ("object_ref",)
    OBJECT_REF_FIELD_NUMBER: _ClassVar[int]
    object_ref: _object_pb2.ObjectRef
    def __init__(self, object_ref: _Optional[_Union[_object_pb2.ObjectRef, _Mapping]] = ...) -> None: ...

class DeleteResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...
