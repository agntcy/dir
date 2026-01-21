from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Iterable as _Iterable, Mapping as _Mapping, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class SyncStatus(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    SYNC_STATUS_UNSPECIFIED: _ClassVar[SyncStatus]
    SYNC_STATUS_PENDING: _ClassVar[SyncStatus]
    SYNC_STATUS_IN_PROGRESS: _ClassVar[SyncStatus]
    SYNC_STATUS_FAILED: _ClassVar[SyncStatus]
    SYNC_STATUS_DELETE_PENDING: _ClassVar[SyncStatus]
    SYNC_STATUS_DELETED: _ClassVar[SyncStatus]
SYNC_STATUS_UNSPECIFIED: SyncStatus
SYNC_STATUS_PENDING: SyncStatus
SYNC_STATUS_IN_PROGRESS: SyncStatus
SYNC_STATUS_FAILED: SyncStatus
SYNC_STATUS_DELETE_PENDING: SyncStatus
SYNC_STATUS_DELETED: SyncStatus

class CreateSyncRequest(_message.Message):
    __slots__ = ("remote_directory_url", "cids")
    REMOTE_DIRECTORY_URL_FIELD_NUMBER: _ClassVar[int]
    CIDS_FIELD_NUMBER: _ClassVar[int]
    remote_directory_url: str
    cids: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, remote_directory_url: _Optional[str] = ..., cids: _Optional[_Iterable[str]] = ...) -> None: ...

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

class ListSyncsItem(_message.Message):
    __slots__ = ("sync_id", "status", "remote_directory_url")
    SYNC_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    REMOTE_DIRECTORY_URL_FIELD_NUMBER: _ClassVar[int]
    sync_id: str
    status: SyncStatus
    remote_directory_url: str
    def __init__(self, sync_id: _Optional[str] = ..., status: _Optional[_Union[SyncStatus, str]] = ..., remote_directory_url: _Optional[str] = ...) -> None: ...

class GetSyncRequest(_message.Message):
    __slots__ = ("sync_id",)
    SYNC_ID_FIELD_NUMBER: _ClassVar[int]
    sync_id: str
    def __init__(self, sync_id: _Optional[str] = ...) -> None: ...

class GetSyncResponse(_message.Message):
    __slots__ = ("sync_id", "status", "remote_directory_url", "created_time", "last_update_time")
    SYNC_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    REMOTE_DIRECTORY_URL_FIELD_NUMBER: _ClassVar[int]
    CREATED_TIME_FIELD_NUMBER: _ClassVar[int]
    LAST_UPDATE_TIME_FIELD_NUMBER: _ClassVar[int]
    sync_id: str
    status: SyncStatus
    remote_directory_url: str
    created_time: str
    last_update_time: str
    def __init__(self, sync_id: _Optional[str] = ..., status: _Optional[_Union[SyncStatus, str]] = ..., remote_directory_url: _Optional[str] = ..., created_time: _Optional[str] = ..., last_update_time: _Optional[str] = ...) -> None: ...

class DeleteSyncRequest(_message.Message):
    __slots__ = ("sync_id",)
    SYNC_ID_FIELD_NUMBER: _ClassVar[int]
    sync_id: str
    def __init__(self, sync_id: _Optional[str] = ...) -> None: ...

class DeleteSyncResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class RequestRegistryCredentialsRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class RequestRegistryCredentialsResponse(_message.Message):
    __slots__ = ("success", "error_message", "registry_address", "repository_name", "basic_auth", "insecure")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    ERROR_MESSAGE_FIELD_NUMBER: _ClassVar[int]
    REGISTRY_ADDRESS_FIELD_NUMBER: _ClassVar[int]
    REPOSITORY_NAME_FIELD_NUMBER: _ClassVar[int]
    BASIC_AUTH_FIELD_NUMBER: _ClassVar[int]
    INSECURE_FIELD_NUMBER: _ClassVar[int]
    success: bool
    error_message: str
    registry_address: str
    repository_name: str
    basic_auth: BasicAuthCredentials
    insecure: bool
    def __init__(self, success: bool = ..., error_message: _Optional[str] = ..., registry_address: _Optional[str] = ..., repository_name: _Optional[str] = ..., basic_auth: _Optional[_Union[BasicAuthCredentials, _Mapping]] = ..., insecure: bool = ...) -> None: ...

class BasicAuthCredentials(_message.Message):
    __slots__ = ("username", "password")
    USERNAME_FIELD_NUMBER: _ClassVar[int]
    PASSWORD_FIELD_NUMBER: _ClassVar[int]
    username: str
    password: str
    def __init__(self, username: _Optional[str] = ..., password: _Optional[str] = ...) -> None: ...
