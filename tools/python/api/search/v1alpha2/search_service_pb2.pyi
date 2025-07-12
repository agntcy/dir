from core.v1 import record_pb2 as _record_pb2
from search.v1alpha2 import record_query_pb2 as _record_query_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class SearchRequest(_message.Message):
    __slots__ = ("queries", "limit", "offset")
    QUERIES_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    OFFSET_FIELD_NUMBER: _ClassVar[int]
    queries: _containers.RepeatedCompositeFieldContainer[_record_query_pb2.RecordQuery]
    limit: int
    offset: int
    def __init__(self, queries: _Optional[_Iterable[_Union[_record_query_pb2.RecordQuery, _Mapping]]] = ..., limit: _Optional[int] = ..., offset: _Optional[int] = ...) -> None: ...

class SearchResponse(_message.Message):
    __slots__ = ("record_ref",)
    RECORD_REF_FIELD_NUMBER: _ClassVar[int]
    record_ref: _record_pb2.RecordRef
    def __init__(self, record_ref: _Optional[_Union[_record_pb2.RecordRef, _Mapping]] = ...) -> None: ...
