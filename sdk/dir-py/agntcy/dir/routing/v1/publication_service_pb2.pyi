from agntcy.dir.routing.v1 import routing_service_pb2 as _routing_service_pb2
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class PublicationStatus(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    PUBLICATION_STATUS_UNSPECIFIED: _ClassVar[PublicationStatus]
    PUBLICATION_STATUS_PENDING: _ClassVar[PublicationStatus]
    PUBLICATION_STATUS_IN_PROGRESS: _ClassVar[PublicationStatus]
    PUBLICATION_STATUS_COMPLETED: _ClassVar[PublicationStatus]
    PUBLICATION_STATUS_FAILED: _ClassVar[PublicationStatus]
PUBLICATION_STATUS_UNSPECIFIED: PublicationStatus
PUBLICATION_STATUS_PENDING: PublicationStatus
PUBLICATION_STATUS_IN_PROGRESS: PublicationStatus
PUBLICATION_STATUS_COMPLETED: PublicationStatus
PUBLICATION_STATUS_FAILED: PublicationStatus

class CreatePublicationResponse(_message.Message):
    __slots__ = ("publication_id",)
    PUBLICATION_ID_FIELD_NUMBER: _ClassVar[int]
    publication_id: str
    def __init__(self, publication_id: _Optional[str] = ...) -> None: ...

class ListPublicationsRequest(_message.Message):
    __slots__ = ("limit", "offset")
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    OFFSET_FIELD_NUMBER: _ClassVar[int]
    limit: int
    offset: int
    def __init__(self, limit: _Optional[int] = ..., offset: _Optional[int] = ...) -> None: ...

class ListPublicationsItem(_message.Message):
    __slots__ = ("publication_id", "status", "created_time", "last_update_time")
    PUBLICATION_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    CREATED_TIME_FIELD_NUMBER: _ClassVar[int]
    LAST_UPDATE_TIME_FIELD_NUMBER: _ClassVar[int]
    publication_id: str
    status: PublicationStatus
    created_time: str
    last_update_time: str
    def __init__(self, publication_id: _Optional[str] = ..., status: _Optional[_Union[PublicationStatus, str]] = ..., created_time: _Optional[str] = ..., last_update_time: _Optional[str] = ...) -> None: ...

class GetPublicationRequest(_message.Message):
    __slots__ = ("publication_id",)
    PUBLICATION_ID_FIELD_NUMBER: _ClassVar[int]
    publication_id: str
    def __init__(self, publication_id: _Optional[str] = ...) -> None: ...

class GetPublicationResponse(_message.Message):
    __slots__ = ("publication_id", "status", "created_time", "last_update_time")
    PUBLICATION_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    CREATED_TIME_FIELD_NUMBER: _ClassVar[int]
    LAST_UPDATE_TIME_FIELD_NUMBER: _ClassVar[int]
    publication_id: str
    status: PublicationStatus
    created_time: str
    last_update_time: str
    def __init__(self, publication_id: _Optional[str] = ..., status: _Optional[_Union[PublicationStatus, str]] = ..., created_time: _Optional[str] = ..., last_update_time: _Optional[str] = ...) -> None: ...
