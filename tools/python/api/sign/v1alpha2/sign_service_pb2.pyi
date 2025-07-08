from core.v1alpha2 import record_pb2 as _record_pb2
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class SignRequest(_message.Message):
    __slots__ = ("record", "provider")
    RECORD_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_FIELD_NUMBER: _ClassVar[int]
    record: _record_pb2.Record
    provider: SignRequestProvider
    def __init__(self, record: _Optional[_Union[_record_pb2.Record, _Mapping]] = ..., provider: _Optional[_Union[SignRequestProvider, _Mapping]] = ...) -> None: ...

class SignRequestProvider(_message.Message):
    __slots__ = ("oidc", "key")
    OIDC_FIELD_NUMBER: _ClassVar[int]
    KEY_FIELD_NUMBER: _ClassVar[int]
    oidc: SignWithOIDC
    key: SignWithKey
    def __init__(self, oidc: _Optional[_Union[SignWithOIDC, _Mapping]] = ..., key: _Optional[_Union[SignWithKey, _Mapping]] = ...) -> None: ...

class SignResponse(_message.Message):
    __slots__ = ("record",)
    RECORD_FIELD_NUMBER: _ClassVar[int]
    record: _record_pb2.Record
    def __init__(self, record: _Optional[_Union[_record_pb2.Record, _Mapping]] = ...) -> None: ...

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

class VerifyRequest(_message.Message):
    __slots__ = ("record", "provider")
    RECORD_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_FIELD_NUMBER: _ClassVar[int]
    record: _record_pb2.Record
    provider: VerifyRequestProvider
    def __init__(self, record: _Optional[_Union[_record_pb2.Record, _Mapping]] = ..., provider: _Optional[_Union[VerifyRequestProvider, _Mapping]] = ...) -> None: ...

class VerifyRequestProvider(_message.Message):
    __slots__ = ("oidc", "key")
    OIDC_FIELD_NUMBER: _ClassVar[int]
    KEY_FIELD_NUMBER: _ClassVar[int]
    oidc: VerifyWithOIDC
    key: VerifyWithKey
    def __init__(self, oidc: _Optional[_Union[VerifyWithOIDC, _Mapping]] = ..., key: _Optional[_Union[VerifyWithKey, _Mapping]] = ...) -> None: ...

class VerifyWithOIDC(_message.Message):
    __slots__ = ("expected_issuer", "expected_signer")
    EXPECTED_ISSUER_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_SIGNER_FIELD_NUMBER: _ClassVar[int]
    expected_issuer: str
    expected_signer: str
    def __init__(self, expected_issuer: _Optional[str] = ..., expected_signer: _Optional[str] = ...) -> None: ...

class VerifyWithKey(_message.Message):
    __slots__ = ("public_key",)
    PUBLIC_KEY_FIELD_NUMBER: _ClassVar[int]
    public_key: bytes
    def __init__(self, public_key: _Optional[bytes] = ...) -> None: ...

class VerifyResponse(_message.Message):
    __slots__ = ("success",)
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    success: bool
    def __init__(self, success: bool = ...) -> None: ...
