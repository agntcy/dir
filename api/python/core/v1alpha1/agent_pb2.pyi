from core.v1alpha1 import extension_pb2 as _extension_pb2
from core.v1alpha1 import locator_pb2 as _locator_pb2
from core.v1alpha1 import skill_pb2 as _skill_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Agent(_message.Message):
    __slots__ = ("schema_version", "name", "version", "description", "authors", "created_at", "annotations", "skills", "locators", "extensions")
    class AnnotationsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    SCHEMA_VERSION_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    AUTHORS_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    ANNOTATIONS_FIELD_NUMBER: _ClassVar[int]
    SKILLS_FIELD_NUMBER: _ClassVar[int]
    LOCATORS_FIELD_NUMBER: _ClassVar[int]
    EXTENSIONS_FIELD_NUMBER: _ClassVar[int]
    schema_version: str
    name: str
    version: str
    description: str
    authors: _containers.RepeatedScalarFieldContainer[str]
    created_at: str
    annotations: _containers.ScalarMap[str, str]
    skills: _containers.RepeatedCompositeFieldContainer[_skill_pb2.Skill]
    locators: _containers.RepeatedCompositeFieldContainer[_locator_pb2.Locator]
    extensions: _containers.RepeatedCompositeFieldContainer[_extension_pb2.Extension]
    def __init__(self, schema_version: _Optional[str] = ..., name: _Optional[str] = ..., version: _Optional[str] = ..., description: _Optional[str] = ..., authors: _Optional[_Iterable[str]] = ..., created_at: _Optional[str] = ..., annotations: _Optional[_Mapping[str, str]] = ..., skills: _Optional[_Iterable[_Union[_skill_pb2.Skill, _Mapping]]] = ..., locators: _Optional[_Iterable[_Union[_locator_pb2.Locator, _Mapping]]] = ..., extensions: _Optional[_Iterable[_Union[_extension_pb2.Extension, _Mapping]]] = ...) -> None: ...
