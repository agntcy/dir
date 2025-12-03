from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Iterable as _Iterable, Mapping as _Mapping, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ObjectRef(_message.Message):
    __slots__ = ("cid",)
    CID_FIELD_NUMBER: _ClassVar[int]
    cid: str
    def __init__(self, cid: _Optional[str] = ...) -> None: ...

class Object(_message.Message):
    __slots__ = ("parent", "links", "annotations", "created_at", "type", "size", "data")
    class AnnotationsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    PARENT_FIELD_NUMBER: _ClassVar[int]
    LINKS_FIELD_NUMBER: _ClassVar[int]
    ANNOTATIONS_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    TYPE_FIELD_NUMBER: _ClassVar[int]
    SIZE_FIELD_NUMBER: _ClassVar[int]
    DATA_FIELD_NUMBER: _ClassVar[int]
    parent: ObjectRef
    links: _containers.RepeatedCompositeFieldContainer[ObjectRef]
    annotations: _containers.ScalarMap[str, str]
    created_at: str
    type: str
    size: int
    data: bytes
    def __init__(self, parent: _Optional[_Union[ObjectRef, _Mapping]] = ..., links: _Optional[_Iterable[_Union[ObjectRef, _Mapping]]] = ..., annotations: _Optional[_Mapping[str, str]] = ..., created_at: _Optional[str] = ..., type: _Optional[str] = ..., size: _Optional[int] = ..., data: _Optional[bytes] = ...) -> None: ...
