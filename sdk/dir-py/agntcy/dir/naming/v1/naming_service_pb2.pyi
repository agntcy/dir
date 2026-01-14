from agntcy.dir.naming.v1 import domain_verification_pb2 as _domain_verification_pb2
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Mapping as _Mapping, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class VerifyDomainRequest(_message.Message):
    __slots__ = ("cid",)
    CID_FIELD_NUMBER: _ClassVar[int]
    cid: str
    def __init__(self, cid: _Optional[str] = ...) -> None: ...

class VerifyDomainResponse(_message.Message):
    __slots__ = ("verified", "verification", "error_message")
    VERIFIED_FIELD_NUMBER: _ClassVar[int]
    VERIFICATION_FIELD_NUMBER: _ClassVar[int]
    ERROR_MESSAGE_FIELD_NUMBER: _ClassVar[int]
    verified: bool
    verification: _domain_verification_pb2.DomainVerification
    error_message: str
    def __init__(self, verified: bool = ..., verification: _Optional[_Union[_domain_verification_pb2.DomainVerification, _Mapping]] = ..., error_message: _Optional[str] = ...) -> None: ...

class CheckDomainVerificationRequest(_message.Message):
    __slots__ = ("cid",)
    CID_FIELD_NUMBER: _ClassVar[int]
    cid: str
    def __init__(self, cid: _Optional[str] = ...) -> None: ...

class CheckDomainVerificationResponse(_message.Message):
    __slots__ = ("verified", "verification", "error_message")
    VERIFIED_FIELD_NUMBER: _ClassVar[int]
    VERIFICATION_FIELD_NUMBER: _ClassVar[int]
    ERROR_MESSAGE_FIELD_NUMBER: _ClassVar[int]
    verified: bool
    verification: _domain_verification_pb2.DomainVerification
    error_message: str
    def __init__(self, verified: bool = ..., verification: _Optional[_Union[_domain_verification_pb2.DomainVerification, _Mapping]] = ..., error_message: _Optional[str] = ...) -> None: ...
