from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional

DESCRIPTOR: _descriptor.FileDescriptor

class AgentSignatureType(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    AGENT_SIGNATURE_TYPE_UNSPECIFIED: _ClassVar[AgentSignatureType]
    AGENT_SIGNATURE_TYPE_SIGSTORE: _ClassVar[AgentSignatureType]
AGENT_SIGNATURE_TYPE_UNSPECIFIED: AgentSignatureType
AGENT_SIGNATURE_TYPE_SIGSTORE: AgentSignatureType

class AgentSignature(_message.Message):
    __slots__ = ("type", "data", "annotations", "signed_at")
    class AnnotationsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    TYPE_FIELD_NUMBER: _ClassVar[int]
    DATA_FIELD_NUMBER: _ClassVar[int]
    ANNOTATIONS_FIELD_NUMBER: _ClassVar[int]
    SIGNED_AT_FIELD_NUMBER: _ClassVar[int]
    type: str
    data: str
    annotations: _containers.ScalarMap[str, str]
    signed_at: str
    def __init__(self, type: _Optional[str] = ..., data: _Optional[str] = ..., annotations: _Optional[_Mapping[str, str]] = ..., signed_at: _Optional[str] = ...) -> None: ...
