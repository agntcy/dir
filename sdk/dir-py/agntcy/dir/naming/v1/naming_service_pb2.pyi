from agntcy.dir.core.v1 import record_pb2 as _record_pb2
from agntcy.dir.naming.v1 import name_verification_pb2 as _name_verification_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Iterable as _Iterable, Mapping as _Mapping, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class GetVerificationInfoRequest(_message.Message):
    __slots__ = ("cid",)
    CID_FIELD_NUMBER: _ClassVar[int]
    cid: str
    def __init__(self, cid: _Optional[str] = ...) -> None: ...

class GetVerificationInfoResponse(_message.Message):
    __slots__ = ("verified", "verification", "error_message")
    VERIFIED_FIELD_NUMBER: _ClassVar[int]
    VERIFICATION_FIELD_NUMBER: _ClassVar[int]
    ERROR_MESSAGE_FIELD_NUMBER: _ClassVar[int]
    verified: bool
    verification: _name_verification_pb2.Verification
    error_message: str
    def __init__(self, verified: bool = ..., verification: _Optional[_Union[_name_verification_pb2.Verification, _Mapping]] = ..., error_message: _Optional[str] = ...) -> None: ...

class ResolveRequest(_message.Message):
    __slots__ = ("name", "version")
    NAME_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    name: str
    version: str
    def __init__(self, name: _Optional[str] = ..., version: _Optional[str] = ...) -> None: ...

class ResolveResponse(_message.Message):
    __slots__ = ("records",)
    RECORDS_FIELD_NUMBER: _ClassVar[int]
    records: _containers.RepeatedCompositeFieldContainer[_record_pb2.NamedRecordRef]
    def __init__(self, records: _Optional[_Iterable[_Union[_record_pb2.NamedRecordRef, _Mapping]]] = ...) -> None: ...
