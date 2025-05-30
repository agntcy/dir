from core.v1alpha1 import agent_pb2 as _agent_pb2
from core.v1alpha2 import record_pb2 as _record_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ObjectSchemaType(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    OBJECT_SCHEMA_TYPE_UNSPECIFIED: _ClassVar[ObjectSchemaType]
    OBJECT_SCHEMA_TYPE_RAW: _ClassVar[ObjectSchemaType]
    OBJECT_SCHEMA_TYPE_RECORD: _ClassVar[ObjectSchemaType]
OBJECT_SCHEMA_TYPE_UNSPECIFIED: ObjectSchemaType
OBJECT_SCHEMA_TYPE_RAW: ObjectSchemaType
OBJECT_SCHEMA_TYPE_RECORD: ObjectSchemaType

class ObjectRef(_message.Message):
    __slots__ = ("cid",)
    CID_FIELD_NUMBER: _ClassVar[int]
    cid: str
    def __init__(self, cid: _Optional[str] = ...) -> None: ...

class Object(_message.Message):
    __slots__ = ("cid", "annotations", "schema_type", "schema_version", "object_data")
    class AnnotationsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    CID_FIELD_NUMBER: _ClassVar[int]
    ANNOTATIONS_FIELD_NUMBER: _ClassVar[int]
    SCHEMA_TYPE_FIELD_NUMBER: _ClassVar[int]
    SCHEMA_VERSION_FIELD_NUMBER: _ClassVar[int]
    OBJECT_DATA_FIELD_NUMBER: _ClassVar[int]
    cid: str
    annotations: _containers.ScalarMap[str, str]
    schema_type: str
    schema_version: str
    object_data: ObjectData
    def __init__(self, cid: _Optional[str] = ..., annotations: _Optional[_Mapping[str, str]] = ..., schema_type: _Optional[str] = ..., schema_version: _Optional[str] = ..., object_data: _Optional[_Union[ObjectData, _Mapping]] = ...) -> None: ...

class ObjectData(_message.Message):
    __slots__ = ("raw", "record")
    RAW_FIELD_NUMBER: _ClassVar[int]
    RECORD_FIELD_NUMBER: _ClassVar[int]
    raw: bytes
    record: RecordObjectData
    def __init__(self, raw: _Optional[bytes] = ..., record: _Optional[_Union[RecordObjectData, _Mapping]] = ...) -> None: ...

class RecordObjectData(_message.Message):
    __slots__ = ("record_v1alpha1", "record_v1alpha2")
    RECORD_V1ALPHA1_FIELD_NUMBER: _ClassVar[int]
    RECORD_V1ALPHA2_FIELD_NUMBER: _ClassVar[int]
    record_v1alpha1: _agent_pb2.Agent
    record_v1alpha2: _record_pb2.Record
    def __init__(self, record_v1alpha1: _Optional[_Union[_agent_pb2.Agent, _Mapping]] = ..., record_v1alpha2: _Optional[_Union[_record_pb2.Record, _Mapping]] = ...) -> None: ...
