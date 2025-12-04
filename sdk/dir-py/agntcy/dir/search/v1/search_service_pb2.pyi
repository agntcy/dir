from agntcy.dir.core.v1 import record_pb2 as _record_pb2
from agntcy.dir.search.v1 import record_query_pb2 as _record_query_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Iterable as _Iterable, Mapping as _Mapping, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class SearchCIDsRequest(_message.Message):
    __slots__ = ("queries", "limit", "offset")
    QUERIES_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    OFFSET_FIELD_NUMBER: _ClassVar[int]
    queries: _containers.RepeatedCompositeFieldContainer[_record_query_pb2.RecordQuery]
    limit: int
    offset: int
    def __init__(self, queries: _Optional[_Iterable[_Union[_record_query_pb2.RecordQuery, _Mapping]]] = ..., limit: _Optional[int] = ..., offset: _Optional[int] = ...) -> None: ...

class SearchRecordsRequest(_message.Message):
    __slots__ = ("queries", "limit", "offset")
    QUERIES_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    OFFSET_FIELD_NUMBER: _ClassVar[int]
    queries: _containers.RepeatedCompositeFieldContainer[_record_query_pb2.RecordQuery]
    limit: int
    offset: int
    def __init__(self, queries: _Optional[_Iterable[_Union[_record_query_pb2.RecordQuery, _Mapping]]] = ..., limit: _Optional[int] = ..., offset: _Optional[int] = ...) -> None: ...

class SearchCIDsResponse(_message.Message):
    __slots__ = ("record_cid",)
    RECORD_CID_FIELD_NUMBER: _ClassVar[int]
    record_cid: str
    def __init__(self, record_cid: _Optional[str] = ...) -> None: ...

class SearchRecordsResponse(_message.Message):
    __slots__ = ("record",)
    RECORD_FIELD_NUMBER: _ClassVar[int]
    record: _record_pb2.Record
    def __init__(self, record: _Optional[_Union[_record_pb2.Record, _Mapping]] = ...) -> None: ...
