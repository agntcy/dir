from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Iterable as _Iterable, Mapping as _Mapping, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class PeerConnectionType(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    PEER_CONNECTION_TYPE_NOT_CONNECTED: _ClassVar[PeerConnectionType]
    PEER_CONNECTION_TYPE_CONNECTED: _ClassVar[PeerConnectionType]
    PEER_CONNECTION_TYPE_CAN_CONNECT: _ClassVar[PeerConnectionType]
    PEER_CONNECTION_TYPE_CANNOT_CONNECT: _ClassVar[PeerConnectionType]
PEER_CONNECTION_TYPE_NOT_CONNECTED: PeerConnectionType
PEER_CONNECTION_TYPE_CONNECTED: PeerConnectionType
PEER_CONNECTION_TYPE_CAN_CONNECT: PeerConnectionType
PEER_CONNECTION_TYPE_CANNOT_CONNECT: PeerConnectionType

class Peer(_message.Message):
    __slots__ = ("id", "addrs", "annotations", "connection")
    class AnnotationsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    ID_FIELD_NUMBER: _ClassVar[int]
    ADDRS_FIELD_NUMBER: _ClassVar[int]
    ANNOTATIONS_FIELD_NUMBER: _ClassVar[int]
    CONNECTION_FIELD_NUMBER: _ClassVar[int]
    id: str
    addrs: _containers.RepeatedScalarFieldContainer[str]
    annotations: _containers.ScalarMap[str, str]
    connection: PeerConnectionType
    def __init__(self, id: _Optional[str] = ..., addrs: _Optional[_Iterable[str]] = ..., annotations: _Optional[_Mapping[str, str]] = ..., connection: _Optional[_Union[PeerConnectionType, str]] = ...) -> None: ...
