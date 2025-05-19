from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Optional as _Optional

DESCRIPTOR: _descriptor.FileDescriptor

class Signature(_message.Message):
    __slots__ = ("algorithm", "certificate", "signature", "tlog_url", "signed_at")
    ALGORITHM_FIELD_NUMBER: _ClassVar[int]
    CERTIFICATE_FIELD_NUMBER: _ClassVar[int]
    SIGNATURE_FIELD_NUMBER: _ClassVar[int]
    TLOG_URL_FIELD_NUMBER: _ClassVar[int]
    SIGNED_AT_FIELD_NUMBER: _ClassVar[int]
    algorithm: str
    certificate: str
    signature: str
    tlog_url: str
    signed_at: str
    def __init__(self, algorithm: _Optional[str] = ..., certificate: _Optional[str] = ..., signature: _Optional[str] = ..., tlog_url: _Optional[str] = ..., signed_at: _Optional[str] = ...) -> None: ...
