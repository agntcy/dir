from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional

DESCRIPTOR: _descriptor.FileDescriptor

class Skill(_message.Message):
    __slots__ = ("annotations", "category_uid", "class_uid", "category_name", "class_name")
    class AnnotationsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    ANNOTATIONS_FIELD_NUMBER: _ClassVar[int]
    CATEGORY_UID_FIELD_NUMBER: _ClassVar[int]
    CLASS_UID_FIELD_NUMBER: _ClassVar[int]
    CATEGORY_NAME_FIELD_NUMBER: _ClassVar[int]
    CLASS_NAME_FIELD_NUMBER: _ClassVar[int]
    annotations: _containers.ScalarMap[str, str]
    category_uid: int
    class_uid: int
    category_name: str
    class_name: str
    def __init__(self, annotations: _Optional[_Mapping[str, str]] = ..., category_uid: _Optional[int] = ..., class_uid: _Optional[int] = ..., category_name: _Optional[str] = ..., class_name: _Optional[str] = ...) -> None: ...
