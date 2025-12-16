from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Mapping as _Mapping, Optional as _Optional

DESCRIPTOR: _descriptor.FileDescriptor

class ObjectRef(_message.Message):
    __slots__ = ("cid",)
    CID_FIELD_NUMBER: _ClassVar[int]
    cid: str
    def __init__(self, cid: _Optional[str] = ...) -> None: ...

class ObjectMeta(_message.Message):
    __slots__ = ("cid", "size", "media_type", "artifact_type", "annotations")
    class AnnotationsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    CID_FIELD_NUMBER: _ClassVar[int]
    SIZE_FIELD_NUMBER: _ClassVar[int]
    MEDIA_TYPE_FIELD_NUMBER: _ClassVar[int]
    ARTIFACT_TYPE_FIELD_NUMBER: _ClassVar[int]
    ANNOTATIONS_FIELD_NUMBER: _ClassVar[int]
    cid: str
    size: int
    media_type: str
    artifact_type: str
    annotations: _containers.ScalarMap[str, str]
    def __init__(self, cid: _Optional[str] = ..., size: _Optional[int] = ..., media_type: _Optional[str] = ..., artifact_type: _Optional[str] = ..., annotations: _Optional[_Mapping[str, str]] = ...) -> None: ...

class Object(_message.Message):
    __slots__ = ("media_type", "data")
    MEDIA_TYPE_FIELD_NUMBER: _ClassVar[int]
    DATA_FIELD_NUMBER: _ClassVar[int]
    media_type: str
    data: bytes
    def __init__(self, media_type: _Optional[str] = ..., data: _Optional[bytes] = ...) -> None: ...
