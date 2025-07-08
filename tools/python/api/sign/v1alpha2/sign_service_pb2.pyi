from core.v1alpha2 import record_pb2 as _record_pb2
from google.protobuf import empty_pb2 as _empty_pb2
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class SignOIDCRequest(_message.Message):
    __slots__ = ("record", "id_token", "options")
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
    RECORD_FIELD_NUMBER: _ClassVar[int]
    ID_TOKEN_FIELD_NUMBER: _ClassVar[int]
    OPTIONS_FIELD_NUMBER: _ClassVar[int]
    record: _record_pb2.Record
    id_token: str
    options: SignOIDCRequest.SignOpts
    def __init__(self, record: _Optional[_Union[_record_pb2.Record, _Mapping]] = ..., id_token: _Optional[str] = ..., options: _Optional[_Union[SignOIDCRequest.SignOpts, _Mapping]] = ...) -> None: ...

class SignWithKeyRequest(_message.Message):
    __slots__ = ("record", "private_key", "password")
    RECORD_FIELD_NUMBER: _ClassVar[int]
    PRIVATE_KEY_FIELD_NUMBER: _ClassVar[int]
    PASSWORD_FIELD_NUMBER: _ClassVar[int]
    record: _record_pb2.Record
    private_key: bytes
    password: bytes
    def __init__(self, record: _Optional[_Union[_record_pb2.Record, _Mapping]] = ..., private_key: _Optional[bytes] = ..., password: _Optional[bytes] = ...) -> None: ...

class SignOIDCResponse(_message.Message):
    __slots__ = ("record",)
    RECORD_FIELD_NUMBER: _ClassVar[int]
    record: _record_pb2.Record
    def __init__(self, record: _Optional[_Union[_record_pb2.Record, _Mapping]] = ...) -> None: ...

class SignWithKeyResponse(_message.Message):
    __slots__ = ("record",)
    RECORD_FIELD_NUMBER: _ClassVar[int]
    record: _record_pb2.Record
    def __init__(self, record: _Optional[_Union[_record_pb2.Record, _Mapping]] = ...) -> None: ...

class VerifyOIDCRequest(_message.Message):
    __slots__ = ("record", "expected_issuer", "expected_signer")
    RECORD_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_ISSUER_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_SIGNER_FIELD_NUMBER: _ClassVar[int]
    record: _record_pb2.Record
    expected_issuer: str
    expected_signer: str
    def __init__(self, record: _Optional[_Union[_record_pb2.Record, _Mapping]] = ..., expected_issuer: _Optional[str] = ..., expected_signer: _Optional[str] = ...) -> None: ...

class VerifyWithKeyRequest(_message.Message):
    __slots__ = ("record", "public_key")
    RECORD_FIELD_NUMBER: _ClassVar[int]
    PUBLIC_KEY_FIELD_NUMBER: _ClassVar[int]
    record: _record_pb2.Record
    public_key: bytes
    def __init__(self, record: _Optional[_Union[_record_pb2.Record, _Mapping]] = ..., public_key: _Optional[bytes] = ...) -> None: ...
