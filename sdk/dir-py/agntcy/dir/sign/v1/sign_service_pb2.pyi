from agntcy.dir.core.v1 import record_pb2 as _record_pb2
from agntcy.dir.sign.v1 import signature_pb2 as _signature_pb2
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Mapping as _Mapping, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class SignRequest(_message.Message):
    __slots__ = ("record_ref", "provider")
    RECORD_REF_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_FIELD_NUMBER: _ClassVar[int]
    record_ref: _record_pb2.RecordRef
    provider: SignRequestProvider
    def __init__(self, record_ref: _Optional[_Union[_record_pb2.RecordRef, _Mapping]] = ..., provider: _Optional[_Union[SignRequestProvider, _Mapping]] = ...) -> None: ...

class SignRequestProvider(_message.Message):
    __slots__ = ("oidc", "key")
    OIDC_FIELD_NUMBER: _ClassVar[int]
    KEY_FIELD_NUMBER: _ClassVar[int]
    oidc: SignWithOIDC
    key: SignWithKey
    def __init__(self, oidc: _Optional[_Union[SignWithOIDC, _Mapping]] = ..., key: _Optional[_Union[SignWithKey, _Mapping]] = ...) -> None: ...

class SignWithOIDC(_message.Message):
    __slots__ = ("id_token", "options")
    class SignOpts(_message.Message):
        __slots__ = ("fulcio_url", "rekor_url", "timestamp_url", "oidc_provider_url")
        FULCIO_URL_FIELD_NUMBER: _ClassVar[int]
        REKOR_URL_FIELD_NUMBER: _ClassVar[int]
        TIMESTAMP_URL_FIELD_NUMBER: _ClassVar[int]
        OIDC_PROVIDER_URL_FIELD_NUMBER: _ClassVar[int]
        fulcio_url: str
        rekor_url: str
        timestamp_url: str
        oidc_provider_url: str
        def __init__(self, fulcio_url: _Optional[str] = ..., rekor_url: _Optional[str] = ..., timestamp_url: _Optional[str] = ..., oidc_provider_url: _Optional[str] = ...) -> None: ...
    ID_TOKEN_FIELD_NUMBER: _ClassVar[int]
    OPTIONS_FIELD_NUMBER: _ClassVar[int]
    id_token: str
    options: SignWithOIDC.SignOpts
    def __init__(self, id_token: _Optional[str] = ..., options: _Optional[_Union[SignWithOIDC.SignOpts, _Mapping]] = ...) -> None: ...

class SignWithKey(_message.Message):
    __slots__ = ("private_key", "password")
    PRIVATE_KEY_FIELD_NUMBER: _ClassVar[int]
    PASSWORD_FIELD_NUMBER: _ClassVar[int]
    private_key: bytes
    password: bytes
    def __init__(self, private_key: _Optional[bytes] = ..., password: _Optional[bytes] = ...) -> None: ...

class SignResponse(_message.Message):
    __slots__ = ("signature",)
    SIGNATURE_FIELD_NUMBER: _ClassVar[int]
    signature: _signature_pb2.Signature
    def __init__(self, signature: _Optional[_Union[_signature_pb2.Signature, _Mapping]] = ...) -> None: ...

class VerifyRequest(_message.Message):
    __slots__ = ("record_ref",)
    RECORD_REF_FIELD_NUMBER: _ClassVar[int]
    record_ref: _record_pb2.RecordRef
    def __init__(self, record_ref: _Optional[_Union[_record_pb2.RecordRef, _Mapping]] = ...) -> None: ...

class VerifyResponse(_message.Message):
    __slots__ = ("success", "error_message")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    ERROR_MESSAGE_FIELD_NUMBER: _ClassVar[int]
    success: bool
    error_message: str
    def __init__(self, success: bool = ..., error_message: _Optional[str] = ...) -> None: ...
