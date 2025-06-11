from core.v1alpha2 import skill_pb2 as _skill_pb2
from core.v1alpha2 import locator_pb2 as _locator_pb2
from core.v1alpha2 import extension_pb2 as _extension_pb2
from core.v1alpha2 import signature_pb2 as _signature_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Record(_message.Message):
    __slots__ = ("annotations", "name", "version", "description", "authors", "created_at", "skills", "locators", "extensions", "signature", "tags", "previous_record_cid")
    class AnnotationsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    ANNOTATIONS_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    AUTHORS_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    SKILLS_FIELD_NUMBER: _ClassVar[int]
    LOCATORS_FIELD_NUMBER: _ClassVar[int]
    EXTENSIONS_FIELD_NUMBER: _ClassVar[int]
    SIGNATURE_FIELD_NUMBER: _ClassVar[int]
    TAGS_FIELD_NUMBER: _ClassVar[int]
    PREVIOUS_RECORD_CID_FIELD_NUMBER: _ClassVar[int]
    annotations: _containers.ScalarMap[str, str]
    name: str
    version: str
    description: str
    authors: _containers.RepeatedScalarFieldContainer[str]
    created_at: str
    skills: _containers.RepeatedCompositeFieldContainer[_skill_pb2.Skill]
    locators: _containers.RepeatedCompositeFieldContainer[_locator_pb2.Locator]
    extensions: _containers.RepeatedCompositeFieldContainer[_extension_pb2.Extension]
    signature: _signature_pb2.Signature
    tags: _containers.RepeatedScalarFieldContainer[str]
    previous_record_cid: str
    def __init__(self, annotations: _Optional[_Mapping[str, str]] = ..., name: _Optional[str] = ..., version: _Optional[str] = ..., description: _Optional[str] = ..., authors: _Optional[_Iterable[str]] = ..., created_at: _Optional[str] = ..., skills: _Optional[_Iterable[_Union[_skill_pb2.Skill, _Mapping]]] = ..., locators: _Optional[_Iterable[_Union[_locator_pb2.Locator, _Mapping]]] = ..., extensions: _Optional[_Iterable[_Union[_extension_pb2.Extension, _Mapping]]] = ..., signature: _Optional[_Union[_signature_pb2.Signature, _Mapping]] = ..., tags: _Optional[_Iterable[str]] = ..., previous_record_cid: _Optional[str] = ...) -> None: ...
