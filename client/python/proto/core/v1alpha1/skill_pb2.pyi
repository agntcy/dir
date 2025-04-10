from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional

DESCRIPTOR: _descriptor.FileDescriptor

class Skill(_message.Message):
    __slots__ = ("version", "category_uid", "class_uid", "annotations", "category_name", "class_name")
    class AnnotationsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    VERSION_FIELD_NUMBER: _ClassVar[int]
    CATEGORY_UID_FIELD_NUMBER: _ClassVar[int]
    CLASS_UID_FIELD_NUMBER: _ClassVar[int]
    ANNOTATIONS_FIELD_NUMBER: _ClassVar[int]
    CATEGORY_NAME_FIELD_NUMBER: _ClassVar[int]
    CLASS_NAME_FIELD_NUMBER: _ClassVar[int]
    version: str
    category_uid: str
    class_uid: str
    annotations: _containers.ScalarMap[str, str]
    category_name: str
    class_name: str
    def __init__(self, version: _Optional[str] = ..., category_uid: _Optional[str] = ..., class_uid: _Optional[str] = ..., annotations: _Optional[_Mapping[str, str]] = ..., category_name: _Optional[str] = ..., class_name: _Optional[str] = ...) -> None: ...
