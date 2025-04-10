from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ConnectionType(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    CONNECTION_TYPE_NOT_CONNECTED: _ClassVar[ConnectionType]
    CONNECTION_TYPE_CONNECTED: _ClassVar[ConnectionType]
    CONNECTION_TYPE_CAN_CONNECT: _ClassVar[ConnectionType]
    CONNECTION_TYPE_CANNOT_CONNECT: _ClassVar[ConnectionType]
CONNECTION_TYPE_NOT_CONNECTED: ConnectionType
CONNECTION_TYPE_CONNECTED: ConnectionType
CONNECTION_TYPE_CAN_CONNECT: ConnectionType
CONNECTION_TYPE_CANNOT_CONNECT: ConnectionType

class Peer(_message.Message):
    __slots__ = ("id", "addrs", "connection")
    ID_FIELD_NUMBER: _ClassVar[int]
    ADDRS_FIELD_NUMBER: _ClassVar[int]
    CONNECTION_FIELD_NUMBER: _ClassVar[int]
    id: str
    addrs: _containers.RepeatedScalarFieldContainer[str]
    connection: ConnectionType
    def __init__(self, id: _Optional[str] = ..., addrs: _Optional[_Iterable[str]] = ..., connection: _Optional[_Union[ConnectionType, str]] = ...) -> None: ...
