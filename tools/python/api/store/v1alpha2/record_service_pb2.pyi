from core.v1alpha2 import record_pb2 as _record_pb2
from store.v1alpha2 import object_pb2 as _object_pb2
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Optional as _Optional

DESCRIPTOR: _descriptor.FileDescriptor

class AgentRecordObjectType(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    AGENT_RECORD_OBJECT_TYPE_UNSPECIFIED: _ClassVar[AgentRecordObjectType]
    AGENT_RECORD_OBJECT_TYPE_OASF_AGNTCY_AGENT_V0_3_JSON: _ClassVar[AgentRecordObjectType]
AGENT_RECORD_OBJECT_TYPE_UNSPECIFIED: AgentRecordObjectType
AGENT_RECORD_OBJECT_TYPE_OASF_AGNTCY_AGENT_V0_3_JSON: AgentRecordObjectType

class ValidateRecordResponse(_message.Message):
    __slots__ = ("is_valid", "error_message")
    IS_VALID_FIELD_NUMBER: _ClassVar[int]
    ERROR_MESSAGE_FIELD_NUMBER: _ClassVar[int]
    is_valid: bool
    error_message: str
    def __init__(self, is_valid: bool = ..., error_message: _Optional[str] = ...) -> None: ...
