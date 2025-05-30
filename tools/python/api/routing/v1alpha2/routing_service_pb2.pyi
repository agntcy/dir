from google.protobuf import empty_pb2 as _empty_pb2
from routing.v1alpha2 import peer_pb2 as _peer_pb2
from routing.v1alpha2 import record_query_pb2 as _record_query_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class PublishRequest(_message.Message):
    __slots__ = ("record_cid",)
    RECORD_CID_FIELD_NUMBER: _ClassVar[int]
    record_cid: str
    def __init__(self, record_cid: _Optional[str] = ...) -> None: ...

class UnpublishRequest(_message.Message):
    __slots__ = ("record_cid",)
    RECORD_CID_FIELD_NUMBER: _ClassVar[int]
    record_cid: str
    def __init__(self, record_cid: _Optional[str] = ...) -> None: ...

class SearchRequest(_message.Message):
    __slots__ = ("queries", "min_match_score", "limit")
    QUERIES_FIELD_NUMBER: _ClassVar[int]
    MIN_MATCH_SCORE_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    queries: _containers.RepeatedCompositeFieldContainer[_record_query_pb2.RecordQuery]
    min_match_score: int
    limit: int
    def __init__(self, queries: _Optional[_Iterable[_Union[_record_query_pb2.RecordQuery, _Mapping]]] = ..., min_match_score: _Optional[int] = ..., limit: _Optional[int] = ...) -> None: ...

class SearchResponse(_message.Message):
    __slots__ = ("record_cid", "peer", "match_queries", "match_score")
    RECORD_CID_FIELD_NUMBER: _ClassVar[int]
    PEER_FIELD_NUMBER: _ClassVar[int]
    MATCH_QUERIES_FIELD_NUMBER: _ClassVar[int]
    MATCH_SCORE_FIELD_NUMBER: _ClassVar[int]
    record_cid: str
    peer: _peer_pb2.Peer
    match_queries: _containers.RepeatedCompositeFieldContainer[_record_query_pb2.RecordQuery]
    match_score: int
    def __init__(self, record_cid: _Optional[str] = ..., peer: _Optional[_Union[_peer_pb2.Peer, _Mapping]] = ..., match_queries: _Optional[_Iterable[_Union[_record_query_pb2.RecordQuery, _Mapping]]] = ..., match_score: _Optional[int] = ...) -> None: ...

class ListRequest(_message.Message):
    __slots__ = ("queries", "limit")
    QUERIES_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    queries: _containers.RepeatedCompositeFieldContainer[_record_query_pb2.RecordQuery]
    limit: int
    def __init__(self, queries: _Optional[_Iterable[_Union[_record_query_pb2.RecordQuery, _Mapping]]] = ..., limit: _Optional[int] = ...) -> None: ...

class ListResponse(_message.Message):
    __slots__ = ("record_cid",)
    RECORD_CID_FIELD_NUMBER: _ClassVar[int]
    record_cid: str
    def __init__(self, record_cid: _Optional[str] = ...) -> None: ...
