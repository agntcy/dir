from oasf.v1alpha1 import agent_pb2 as _agent_pb2
from oasf.v1alpha2 import record_pb2 as _record_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class RecordRef(_message.Message):
    __slots__ = ("cid",)
    CID_FIELD_NUMBER: _ClassVar[int]
    cid: str
    def __init__(self, cid: _Optional[str] = ...) -> None: ...

class RecordMeta(_message.Message):
    __slots__ = ("cid", "annotations", "schema_version", "created_at")
    class AnnotationsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    CID_FIELD_NUMBER: _ClassVar[int]
    ANNOTATIONS_FIELD_NUMBER: _ClassVar[int]
    SCHEMA_VERSION_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    cid: str
    annotations: _containers.ScalarMap[str, str]
    schema_version: str
    created_at: str
    def __init__(self, cid: _Optional[str] = ..., annotations: _Optional[_Mapping[str, str]] = ..., schema_version: _Optional[str] = ..., created_at: _Optional[str] = ...) -> None: ...

class Record(_message.Message):
    __slots__ = ("v1alpha1", "v1alpha2")
    V1ALPHA1_FIELD_NUMBER: _ClassVar[int]
    V1ALPHA2_FIELD_NUMBER: _ClassVar[int]
    v1alpha1: _agent_pb2.Agent
    v1alpha2: _record_pb2.Record
    def __init__(self, v1alpha1: _Optional[_Union[_agent_pb2.Agent, _Mapping]] = ..., v1alpha2: _Optional[_Union[_record_pb2.Record, _Mapping]] = ...) -> None: ...
