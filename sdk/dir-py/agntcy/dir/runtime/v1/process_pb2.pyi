from agntcy.dir.core.v1 import record_pb2 as _record_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Mapping as _Mapping, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Process(_message.Message):
    __slots__ = ("pid", "runtime", "created_at", "annotations", "record")
    class AnnotationsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    PID_FIELD_NUMBER: _ClassVar[int]
    RUNTIME_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    ANNOTATIONS_FIELD_NUMBER: _ClassVar[int]
    RECORD_FIELD_NUMBER: _ClassVar[int]
    pid: str
    runtime: str
    created_at: str
    annotations: _containers.ScalarMap[str, str]
    record: _record_pb2.RecordMeta
    def __init__(self, pid: _Optional[str] = ..., runtime: _Optional[str] = ..., created_at: _Optional[str] = ..., annotations: _Optional[_Mapping[str, str]] = ..., record: _Optional[_Union[_record_pb2.RecordMeta, _Mapping]] = ...) -> None: ...
