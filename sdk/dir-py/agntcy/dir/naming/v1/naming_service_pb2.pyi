from agntcy.dir.naming.v1 import name_verification_pb2 as _name_verification_pb2
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Mapping as _Mapping, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class VerifyRequest(_message.Message):
    __slots__ = ("cid",)
    CID_FIELD_NUMBER: _ClassVar[int]
    cid: str
    def __init__(self, cid: _Optional[str] = ...) -> None: ...

class VerifyResponse(_message.Message):
    __slots__ = ("verified", "verification", "error_message")
    VERIFIED_FIELD_NUMBER: _ClassVar[int]
    VERIFICATION_FIELD_NUMBER: _ClassVar[int]
    ERROR_MESSAGE_FIELD_NUMBER: _ClassVar[int]
    verified: bool
    verification: _name_verification_pb2.Verification
    error_message: str
    def __init__(self, verified: bool = ..., verification: _Optional[_Union[_name_verification_pb2.Verification, _Mapping]] = ..., error_message: _Optional[str] = ...) -> None: ...

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
