from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Iterable as _Iterable, Mapping as _Mapping, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class EventType(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    EVENT_TYPE_UNSPECIFIED: _ClassVar[EventType]
    EVENT_TYPE_RECORD_PUSHED: _ClassVar[EventType]
    EVENT_TYPE_RECORD_PULLED: _ClassVar[EventType]
    EVENT_TYPE_RECORD_DELETED: _ClassVar[EventType]
    EVENT_TYPE_RECORD_PUBLISHED: _ClassVar[EventType]
    EVENT_TYPE_RECORD_UNPUBLISHED: _ClassVar[EventType]
    EVENT_TYPE_SYNC_CREATED: _ClassVar[EventType]
    EVENT_TYPE_SYNC_COMPLETED: _ClassVar[EventType]
    EVENT_TYPE_SYNC_FAILED: _ClassVar[EventType]
    EVENT_TYPE_RECORD_SIGNED: _ClassVar[EventType]
EVENT_TYPE_UNSPECIFIED: EventType
EVENT_TYPE_RECORD_PUSHED: EventType
EVENT_TYPE_RECORD_PULLED: EventType
EVENT_TYPE_RECORD_DELETED: EventType
EVENT_TYPE_RECORD_PUBLISHED: EventType
EVENT_TYPE_RECORD_UNPUBLISHED: EventType
EVENT_TYPE_SYNC_CREATED: EventType
EVENT_TYPE_SYNC_COMPLETED: EventType
EVENT_TYPE_SYNC_FAILED: EventType
EVENT_TYPE_RECORD_SIGNED: EventType

class ListenRequest(_message.Message):
    __slots__ = ("event_types", "label_filters", "cid_filters")
    EVENT_TYPES_FIELD_NUMBER: _ClassVar[int]
    LABEL_FILTERS_FIELD_NUMBER: _ClassVar[int]
    CID_FILTERS_FIELD_NUMBER: _ClassVar[int]
    event_types: _containers.RepeatedScalarFieldContainer[EventType]
    label_filters: _containers.RepeatedScalarFieldContainer[str]
    cid_filters: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, event_types: _Optional[_Iterable[_Union[EventType, str]]] = ..., label_filters: _Optional[_Iterable[str]] = ..., cid_filters: _Optional[_Iterable[str]] = ...) -> None: ...

class ListenResponse(_message.Message):
    __slots__ = ("event",)
    EVENT_FIELD_NUMBER: _ClassVar[int]
    event: Event
    def __init__(self, event: _Optional[_Union[Event, _Mapping]] = ...) -> None: ...

class Event(_message.Message):
    __slots__ = ("id", "type", "timestamp", "resource_id", "labels", "metadata")
    class MetadataEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    ID_FIELD_NUMBER: _ClassVar[int]
    TYPE_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    RESOURCE_ID_FIELD_NUMBER: _ClassVar[int]
    LABELS_FIELD_NUMBER: _ClassVar[int]
    METADATA_FIELD_NUMBER: _ClassVar[int]
    id: str
    type: EventType
    timestamp: _timestamp_pb2.Timestamp
    resource_id: str
    labels: _containers.RepeatedScalarFieldContainer[str]
    metadata: _containers.ScalarMap[str, str]
    def __init__(self, id: _Optional[str] = ..., type: _Optional[_Union[EventType, str]] = ..., timestamp: _Optional[_Union[_timestamp_pb2.Timestamp, _Mapping]] = ..., resource_id: _Optional[str] = ..., labels: _Optional[_Iterable[str]] = ..., metadata: _Optional[_Mapping[str, str]] = ...) -> None: ...
