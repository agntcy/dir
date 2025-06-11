from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional

DESCRIPTOR: _descriptor.FileDescriptor

class LocatorType(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    LOCATOR_TYPE_UNSPECIFIED: _ClassVar[LocatorType]
    LOCATOR_TYPE_HELM_CHART: _ClassVar[LocatorType]
    LOCATOR_TYPE_DOCKER_IMAGE: _ClassVar[LocatorType]
    LOCATOR_TYPE_PYTHON_PACKAGE: _ClassVar[LocatorType]
    LOCATOR_TYPE_SOURCE_CODE: _ClassVar[LocatorType]
    LOCATOR_TYPE_BINARY: _ClassVar[LocatorType]
LOCATOR_TYPE_UNSPECIFIED: LocatorType
LOCATOR_TYPE_HELM_CHART: LocatorType
LOCATOR_TYPE_DOCKER_IMAGE: LocatorType
LOCATOR_TYPE_PYTHON_PACKAGE: LocatorType
LOCATOR_TYPE_SOURCE_CODE: LocatorType
LOCATOR_TYPE_BINARY: LocatorType

class Locator(_message.Message):
    __slots__ = ("annotations", "type", "url", "size", "digest")
    class AnnotationsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    ANNOTATIONS_FIELD_NUMBER: _ClassVar[int]
    TYPE_FIELD_NUMBER: _ClassVar[int]
    URL_FIELD_NUMBER: _ClassVar[int]
    SIZE_FIELD_NUMBER: _ClassVar[int]
    DIGEST_FIELD_NUMBER: _ClassVar[int]
    annotations: _containers.ScalarMap[str, str]
    type: str
    url: str
    size: int
    digest: str
    def __init__(self, annotations: _Optional[_Mapping[str, str]] = ..., type: _Optional[str] = ..., url: _Optional[str] = ..., size: _Optional[int] = ..., digest: _Optional[str] = ...) -> None: ...
