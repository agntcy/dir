from agntcy.dir.core.v1 import record_pb2 as _record_pb2
from agntcy.dir.routing.v1 import peer_pb2 as _peer_pb2
from agntcy.dir.routing.v1 import record_query_pb2 as _record_query_pb2
from agntcy.dir.search.v1 import record_query_pb2 as _record_query_pb2_1
from google.protobuf import empty_pb2 as _empty_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Iterable as _Iterable, Mapping as _Mapping, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class PublishRequest(_message.Message):
    __slots__ = ("record_refs", "queries")
    RECORD_REFS_FIELD_NUMBER: _ClassVar[int]
    QUERIES_FIELD_NUMBER: _ClassVar[int]
    record_refs: RecordRefs
    queries: RecordQueries
    def __init__(self, record_refs: _Optional[_Union[RecordRefs, _Mapping]] = ..., queries: _Optional[_Union[RecordQueries, _Mapping]] = ...) -> None: ...

class UnpublishRequest(_message.Message):
    __slots__ = ("record_refs", "queries")
    RECORD_REFS_FIELD_NUMBER: _ClassVar[int]
    QUERIES_FIELD_NUMBER: _ClassVar[int]
    record_refs: RecordRefs
    queries: RecordQueries
    def __init__(self, record_refs: _Optional[_Union[RecordRefs, _Mapping]] = ..., queries: _Optional[_Union[RecordQueries, _Mapping]] = ...) -> None: ...

class RecordRefs(_message.Message):
    __slots__ = ("refs",)
    REFS_FIELD_NUMBER: _ClassVar[int]
    refs: _containers.RepeatedCompositeFieldContainer[_record_pb2.RecordRef]
    def __init__(self, refs: _Optional[_Iterable[_Union[_record_pb2.RecordRef, _Mapping]]] = ...) -> None: ...

class RecordQueries(_message.Message):
    __slots__ = ("queries",)
    QUERIES_FIELD_NUMBER: _ClassVar[int]
    queries: _containers.RepeatedCompositeFieldContainer[_record_query_pb2_1.RecordQuery]
    def __init__(self, queries: _Optional[_Iterable[_Union[_record_query_pb2_1.RecordQuery, _Mapping]]] = ...) -> None: ...

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
    __slots__ = ("record_ref", "peer", "match_queries", "match_score")
    RECORD_REF_FIELD_NUMBER: _ClassVar[int]
    PEER_FIELD_NUMBER: _ClassVar[int]
    MATCH_QUERIES_FIELD_NUMBER: _ClassVar[int]
    MATCH_SCORE_FIELD_NUMBER: _ClassVar[int]
    record_ref: _record_pb2.RecordRef
    peer: _peer_pb2.Peer
    match_queries: _containers.RepeatedCompositeFieldContainer[_record_query_pb2.RecordQuery]
    match_score: int
    def __init__(self, record_ref: _Optional[_Union[_record_pb2.RecordRef, _Mapping]] = ..., peer: _Optional[_Union[_peer_pb2.Peer, _Mapping]] = ..., match_queries: _Optional[_Iterable[_Union[_record_query_pb2.RecordQuery, _Mapping]]] = ..., match_score: _Optional[int] = ...) -> None: ...

class ListRequest(_message.Message):
    __slots__ = ("queries", "limit")
    QUERIES_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    queries: _containers.RepeatedCompositeFieldContainer[_record_query_pb2.RecordQuery]
    limit: int
    def __init__(self, queries: _Optional[_Iterable[_Union[_record_query_pb2.RecordQuery, _Mapping]]] = ..., limit: _Optional[int] = ...) -> None: ...

class ListResponse(_message.Message):
    __slots__ = ("record_ref", "labels")
    RECORD_REF_FIELD_NUMBER: _ClassVar[int]
    LABELS_FIELD_NUMBER: _ClassVar[int]
    record_ref: _record_pb2.RecordRef
    labels: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, record_ref: _Optional[_Union[_record_pb2.RecordRef, _Mapping]] = ..., labels: _Optional[_Iterable[str]] = ...) -> None: ...
