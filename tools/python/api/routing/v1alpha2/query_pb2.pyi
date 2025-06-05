from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class AgentRecordQuery(_message.Message):
    __slots__ = ("type", "value")
    class Type(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
        __slots__ = ()
        TYPE_UNSPECIFIED: _ClassVar[AgentRecordQuery.Type]
        TYPE_SKILL: _ClassVar[AgentRecordQuery.Type]
        TYPE_LOCATOR: _ClassVar[AgentRecordQuery.Type]
    TYPE_UNSPECIFIED: AgentRecordQuery.Type
    TYPE_SKILL: AgentRecordQuery.Type
    TYPE_LOCATOR: AgentRecordQuery.Type
    TYPE_FIELD_NUMBER: _ClassVar[int]
    VALUE_FIELD_NUMBER: _ClassVar[int]
    type: AgentRecordQuery.Type
    value: str
    def __init__(self, type: _Optional[_Union[AgentRecordQuery.Type, str]] = ..., value: _Optional[str] = ...) -> None: ...
