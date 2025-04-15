from core.v1alpha1 import agent_pb2 as _agent_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ObjectType(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    OBJECT_TYPE_RAW: _ClassVar[ObjectType]
    OBJECT_TYPE_AGENT: _ClassVar[ObjectType]
OBJECT_TYPE_RAW: ObjectType
OBJECT_TYPE_AGENT: ObjectType

class ObjectRef(_message.Message):
    __slots__ = ("digest", "type", "size", "annotations")
    class AnnotationsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    DIGEST_FIELD_NUMBER: _ClassVar[int]
    TYPE_FIELD_NUMBER: _ClassVar[int]
    SIZE_FIELD_NUMBER: _ClassVar[int]
    ANNOTATIONS_FIELD_NUMBER: _ClassVar[int]
    digest: str
    type: str
    size: int
    annotations: _containers.ScalarMap[str, str]
    def __init__(self, digest: _Optional[str] = ..., type: _Optional[str] = ..., size: _Optional[int] = ..., annotations: _Optional[_Mapping[str, str]] = ...) -> None: ...

class Object(_message.Message):
    __slots__ = ("data", "ref", "agent")
    DATA_FIELD_NUMBER: _ClassVar[int]
    REF_FIELD_NUMBER: _ClassVar[int]
    AGENT_FIELD_NUMBER: _ClassVar[int]
    data: bytes
    ref: ObjectRef
    agent: _agent_pb2.Agent
    def __init__(self, data: _Optional[bytes] = ..., ref: _Optional[_Union[ObjectRef, _Mapping]] = ..., agent: _Optional[_Union[_agent_pb2.Agent, _Mapping]] = ...) -> None: ...
