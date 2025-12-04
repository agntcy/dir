from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class RecordQueryType(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    RECORD_QUERY_TYPE_UNSPECIFIED: _ClassVar[RecordQueryType]
    RECORD_QUERY_TYPE_NAME: _ClassVar[RecordQueryType]
    RECORD_QUERY_TYPE_VERSION: _ClassVar[RecordQueryType]
    RECORD_QUERY_TYPE_SKILL_ID: _ClassVar[RecordQueryType]
    RECORD_QUERY_TYPE_SKILL_NAME: _ClassVar[RecordQueryType]
    RECORD_QUERY_TYPE_LOCATOR: _ClassVar[RecordQueryType]
    RECORD_QUERY_TYPE_MODULE_NAME: _ClassVar[RecordQueryType]
    RECORD_QUERY_TYPE_DOMAIN_ID: _ClassVar[RecordQueryType]
    RECORD_QUERY_TYPE_DOMAIN_NAME: _ClassVar[RecordQueryType]
    RECORD_QUERY_TYPE_CREATED_AT: _ClassVar[RecordQueryType]
    RECORD_QUERY_TYPE_AUTHOR: _ClassVar[RecordQueryType]
    RECORD_QUERY_TYPE_SCHEMA_VERSION: _ClassVar[RecordQueryType]
    RECORD_QUERY_TYPE_MODULE_ID: _ClassVar[RecordQueryType]
RECORD_QUERY_TYPE_UNSPECIFIED: RecordQueryType
RECORD_QUERY_TYPE_NAME: RecordQueryType
RECORD_QUERY_TYPE_VERSION: RecordQueryType
RECORD_QUERY_TYPE_SKILL_ID: RecordQueryType
RECORD_QUERY_TYPE_SKILL_NAME: RecordQueryType
RECORD_QUERY_TYPE_LOCATOR: RecordQueryType
RECORD_QUERY_TYPE_MODULE_NAME: RecordQueryType
RECORD_QUERY_TYPE_DOMAIN_ID: RecordQueryType
RECORD_QUERY_TYPE_DOMAIN_NAME: RecordQueryType
RECORD_QUERY_TYPE_CREATED_AT: RecordQueryType
RECORD_QUERY_TYPE_AUTHOR: RecordQueryType
RECORD_QUERY_TYPE_SCHEMA_VERSION: RecordQueryType
RECORD_QUERY_TYPE_MODULE_ID: RecordQueryType

class RecordQuery(_message.Message):
    __slots__ = ("type", "value")
    TYPE_FIELD_NUMBER: _ClassVar[int]
    VALUE_FIELD_NUMBER: _ClassVar[int]
    type: RecordQueryType
    value: str
    def __init__(self, type: _Optional[_Union[RecordQueryType, str]] = ..., value: _Optional[str] = ...) -> None: ...
