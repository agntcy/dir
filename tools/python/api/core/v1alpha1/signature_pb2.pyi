from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Optional as _Optional

DESCRIPTOR: _descriptor.FileDescriptor

class Signature(_message.Message):
    __slots__ = ("algorithm", "signature", "certificate", "content_type", "content_bundle", "signed_at")
    ALGORITHM_FIELD_NUMBER: _ClassVar[int]
    SIGNATURE_FIELD_NUMBER: _ClassVar[int]
    CERTIFICATE_FIELD_NUMBER: _ClassVar[int]
    CONTENT_TYPE_FIELD_NUMBER: _ClassVar[int]
    CONTENT_BUNDLE_FIELD_NUMBER: _ClassVar[int]
    SIGNED_AT_FIELD_NUMBER: _ClassVar[int]
    algorithm: str
    signature: str
    certificate: str
    content_type: str
    content_bundle: str
    signed_at: str
    def __init__(self, algorithm: _Optional[str] = ..., signature: _Optional[str] = ..., certificate: _Optional[str] = ..., content_type: _Optional[str] = ..., content_bundle: _Optional[str] = ..., signed_at: _Optional[str] = ...) -> None: ...
