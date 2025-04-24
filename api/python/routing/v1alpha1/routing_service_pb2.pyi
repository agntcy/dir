from google.protobuf import empty_pb2 as _empty_pb2
from core.v1alpha1 import object_pb2 as _object_pb2
from routing.v1alpha1 import peer_pb2 as _peer_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class PublishRequest(_message.Message):
    __slots__ = ("record", "network")
    RECORD_FIELD_NUMBER: _ClassVar[int]
    NETWORK_FIELD_NUMBER: _ClassVar[int]
    record: _object_pb2.ObjectRef
    network: bool
    def __init__(self, record: _Optional[_Union[_object_pb2.ObjectRef, _Mapping]] = ..., network: bool = ...) -> None: ...

class ListRequest(_message.Message):
    __slots__ = ("peer", "labels", "record", "max_hops", "network")
    PEER_FIELD_NUMBER: _ClassVar[int]
    LABELS_FIELD_NUMBER: _ClassVar[int]
    RECORD_FIELD_NUMBER: _ClassVar[int]
    MAX_HOPS_FIELD_NUMBER: _ClassVar[int]
    NETWORK_FIELD_NUMBER: _ClassVar[int]
    peer: _peer_pb2.Peer
    labels: _containers.RepeatedScalarFieldContainer[str]
    record: _object_pb2.ObjectRef
    max_hops: int
    network: bool
    def __init__(self, peer: _Optional[_Union[_peer_pb2.Peer, _Mapping]] = ..., labels: _Optional[_Iterable[str]] = ..., record: _Optional[_Union[_object_pb2.ObjectRef, _Mapping]] = ..., max_hops: _Optional[int] = ..., network: bool = ...) -> None: ...

class ListResponse(_message.Message):
    __slots__ = ("items",)
    class Item(_message.Message):
        __slots__ = ("labels", "label_counts", "peer", "record")
        class LabelCountsEntry(_message.Message):
            __slots__ = ("key", "value")
            KEY_FIELD_NUMBER: _ClassVar[int]
            VALUE_FIELD_NUMBER: _ClassVar[int]
            key: str
            value: int
            def __init__(self, key: _Optional[str] = ..., value: _Optional[int] = ...) -> None: ...
        LABELS_FIELD_NUMBER: _ClassVar[int]
        LABEL_COUNTS_FIELD_NUMBER: _ClassVar[int]
        PEER_FIELD_NUMBER: _ClassVar[int]
        RECORD_FIELD_NUMBER: _ClassVar[int]
        labels: _containers.RepeatedScalarFieldContainer[str]
        label_counts: _containers.ScalarMap[str, int]
        peer: _peer_pb2.Peer
        record: _object_pb2.ObjectRef
        def __init__(self, labels: _Optional[_Iterable[str]] = ..., label_counts: _Optional[_Mapping[str, int]] = ..., peer: _Optional[_Union[_peer_pb2.Peer, _Mapping]] = ..., record: _Optional[_Union[_object_pb2.ObjectRef, _Mapping]] = ...) -> None: ...
    ITEMS_FIELD_NUMBER: _ClassVar[int]
    items: _containers.RepeatedCompositeFieldContainer[ListResponse.Item]
    def __init__(self, items: _Optional[_Iterable[_Union[ListResponse.Item, _Mapping]]] = ...) -> None: ...

class UnpublishRequest(_message.Message):
    __slots__ = ("record", "network")
    RECORD_FIELD_NUMBER: _ClassVar[int]
    NETWORK_FIELD_NUMBER: _ClassVar[int]
    record: _object_pb2.ObjectRef
    network: bool
    def __init__(self, record: _Optional[_Union[_object_pb2.ObjectRef, _Mapping]] = ..., network: bool = ...) -> None: ...
