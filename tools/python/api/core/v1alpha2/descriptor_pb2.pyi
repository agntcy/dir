from google.protobuf import struct_pb2 as _struct_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class AgentDescriptor(_message.Message):
    __slots__ = ("name", "annotations", "descriptor_data")
    class AnnotationsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    NAME_FIELD_NUMBER: _ClassVar[int]
    ANNOTATIONS_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTOR_DATA_FIELD_NUMBER: _ClassVar[int]
    name: str
    annotations: _containers.ScalarMap[str, str]
    descriptor_data: _struct_pb2.Struct
    def __init__(self, name: _Optional[str] = ..., annotations: _Optional[_Mapping[str, str]] = ..., descriptor_data: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ...) -> None: ...
