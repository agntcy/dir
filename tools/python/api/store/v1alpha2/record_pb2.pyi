from store.v1alpha2 import object_pb2 as _object_pb2
from core.v1alpha1 import agent_pb2 as _agent_pb2
from core.v1alpha2 import record_pb2 as _record_pb2
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class RecordObjectType(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    RECORD_OBJECT_TYPE_UNSPECIFIED: _ClassVar[RecordObjectType]
    RECORD_OBJECT_TYPE_OASF_V1ALPHA1_JSON: _ClassVar[RecordObjectType]
    RECORD_OBJECT_TYPE_OASF_V1ALPHA2_JSON: _ClassVar[RecordObjectType]
RECORD_OBJECT_TYPE_UNSPECIFIED: RecordObjectType
RECORD_OBJECT_TYPE_OASF_V1ALPHA1_JSON: RecordObjectType
RECORD_OBJECT_TYPE_OASF_V1ALPHA2_JSON: RecordObjectType

class RecordObject(_message.Message):
    __slots__ = ("cid", "type", "record")
    CID_FIELD_NUMBER: _ClassVar[int]
    TYPE_FIELD_NUMBER: _ClassVar[int]
    RECORD_FIELD_NUMBER: _ClassVar[int]
    cid: str
    type: RecordObjectType
    record: RecordObjectData
    def __init__(self, cid: _Optional[str] = ..., type: _Optional[_Union[RecordObjectType, str]] = ..., record: _Optional[_Union[RecordObjectData, _Mapping]] = ...) -> None: ...

class RecordObjectData(_message.Message):
    __slots__ = ("record_v1alpha1", "record_v1alpha2")
    RECORD_V1ALPHA1_FIELD_NUMBER: _ClassVar[int]
    RECORD_V1ALPHA2_FIELD_NUMBER: _ClassVar[int]
    record_v1alpha1: _agent_pb2.Agent
    record_v1alpha2: _record_pb2.Record
    def __init__(self, record_v1alpha1: _Optional[_Union[_agent_pb2.Agent, _Mapping]] = ..., record_v1alpha2: _Optional[_Union[_record_pb2.Record, _Mapping]] = ...) -> None: ...
