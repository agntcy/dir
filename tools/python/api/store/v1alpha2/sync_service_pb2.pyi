from google.protobuf import empty_pb2 as _empty_pb2
from store.v1alpha2 import object_pb2 as _object_pb2
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class SyncStatus(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    SYNC_STATUS_UNSPECIFIED: _ClassVar[SyncStatus]
    SYNC_STATUS_PENDING: _ClassVar[SyncStatus]
    SYNC_STATUS_IN_PROGRESS: _ClassVar[SyncStatus]
    SYNC_STATUS_COMPLETED: _ClassVar[SyncStatus]
    SYNC_STATUS_FAILED: _ClassVar[SyncStatus]
SYNC_STATUS_UNSPECIFIED: SyncStatus
SYNC_STATUS_PENDING: SyncStatus
SYNC_STATUS_IN_PROGRESS: SyncStatus
SYNC_STATUS_COMPLETED: SyncStatus
SYNC_STATUS_FAILED: SyncStatus

class CreateSyncRequest(_message.Message):
    __slots__ = ("remote_directory",)
    REMOTE_DIRECTORY_FIELD_NUMBER: _ClassVar[int]
    remote_directory: str
    def __init__(self, remote_directory: _Optional[str] = ...) -> None: ...

class CreateSyncResponse(_message.Message):
    __slots__ = ("sync_id",)
    SYNC_ID_FIELD_NUMBER: _ClassVar[int]
    sync_id: str
    def __init__(self, sync_id: _Optional[str] = ...) -> None: ...

class ListSyncsRequest(_message.Message):
    __slots__ = ("limit", "offset")
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    OFFSET_FIELD_NUMBER: _ClassVar[int]
    limit: int
    offset: int
    def __init__(self, limit: _Optional[int] = ..., offset: _Optional[int] = ...) -> None: ...

class ListSyncItem(_message.Message):
    __slots__ = ("sync_id", "status", "remote_directory")
    SYNC_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    REMOTE_DIRECTORY_FIELD_NUMBER: _ClassVar[int]
    sync_id: str
    status: str
    remote_directory: str
    def __init__(self, sync_id: _Optional[str] = ..., status: _Optional[str] = ..., remote_directory: _Optional[str] = ...) -> None: ...

class GetSyncRequest(_message.Message):
    __slots__ = ("sync_id",)
    SYNC_ID_FIELD_NUMBER: _ClassVar[int]
    sync_id: str
    def __init__(self, sync_id: _Optional[str] = ...) -> None: ...

class GetSyncResponse(_message.Message):
    __slots__ = ("id", "status", "remote_directory", "created_time", "last_update_time")
    ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    REMOTE_DIRECTORY_FIELD_NUMBER: _ClassVar[int]
    CREATED_TIME_FIELD_NUMBER: _ClassVar[int]
    LAST_UPDATE_TIME_FIELD_NUMBER: _ClassVar[int]
    id: str
    status: SyncStatus
    remote_directory: str
    created_time: str
    last_update_time: str
    def __init__(self, id: _Optional[str] = ..., status: _Optional[_Union[SyncStatus, str]] = ..., remote_directory: _Optional[str] = ..., created_time: _Optional[str] = ..., last_update_time: _Optional[str] = ...) -> None: ...

class DeleteSyncRequest(_message.Message):
    __slots__ = ("sync_id",)
    SYNC_ID_FIELD_NUMBER: _ClassVar[int]
    sync_id: str
    def __init__(self, sync_id: _Optional[str] = ...) -> None: ...
