from agntcy.dir.core.v1 import record_pb2 as _record_pb2
from agntcy.dir.sign.v1 import signature_pb2 as _signature_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Iterable as _Iterable, Mapping as _Mapping, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class SignOptionsOIDC(_message.Message):
    __slots__ = ("fulcio_url", "rekor_url", "timestamp_url", "oidc_provider_url", "oidc_client_id", "oidc_client_secret", "skip_tlog")
    FULCIO_URL_FIELD_NUMBER: _ClassVar[int]
    REKOR_URL_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_URL_FIELD_NUMBER: _ClassVar[int]
    OIDC_PROVIDER_URL_FIELD_NUMBER: _ClassVar[int]
    OIDC_CLIENT_ID_FIELD_NUMBER: _ClassVar[int]
    OIDC_CLIENT_SECRET_FIELD_NUMBER: _ClassVar[int]
    SKIP_TLOG_FIELD_NUMBER: _ClassVar[int]
    fulcio_url: str
    rekor_url: str
    timestamp_url: str
    oidc_provider_url: str
    oidc_client_id: str
    oidc_client_secret: str
    skip_tlog: bool
    def __init__(self, fulcio_url: _Optional[str] = ..., rekor_url: _Optional[str] = ..., timestamp_url: _Optional[str] = ..., oidc_provider_url: _Optional[str] = ..., oidc_client_id: _Optional[str] = ..., oidc_client_secret: _Optional[str] = ..., skip_tlog: bool = ...) -> None: ...

class VerifyOptionsOIDC(_message.Message):
    __slots__ = ("tuf_mirror_url", "trusted_root_path", "ignore_tlog", "ignore_tsa", "ignore_sct")
    TUF_MIRROR_URL_FIELD_NUMBER: _ClassVar[int]
    TRUSTED_ROOT_PATH_FIELD_NUMBER: _ClassVar[int]
    IGNORE_TLOG_FIELD_NUMBER: _ClassVar[int]
    IGNORE_TSA_FIELD_NUMBER: _ClassVar[int]
    IGNORE_SCT_FIELD_NUMBER: _ClassVar[int]
    tuf_mirror_url: str
    trusted_root_path: str
    ignore_tlog: bool
    ignore_tsa: bool
    ignore_sct: bool
    def __init__(self, tuf_mirror_url: _Optional[str] = ..., trusted_root_path: _Optional[str] = ..., ignore_tlog: bool = ..., ignore_tsa: bool = ..., ignore_sct: bool = ...) -> None: ...

class SignRequest(_message.Message):
    __slots__ = ("record_ref", "provider")
    RECORD_REF_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_FIELD_NUMBER: _ClassVar[int]
    record_ref: _record_pb2.RecordRef
    provider: SignRequestProvider
    def __init__(self, record_ref: _Optional[_Union[_record_pb2.RecordRef, _Mapping]] = ..., provider: _Optional[_Union[SignRequestProvider, _Mapping]] = ...) -> None: ...

class SignRequestProvider(_message.Message):
    __slots__ = ("key", "oidc")
    KEY_FIELD_NUMBER: _ClassVar[int]
    OIDC_FIELD_NUMBER: _ClassVar[int]
    key: SignWithKey
    oidc: SignWithOIDC
    def __init__(self, key: _Optional[_Union[SignWithKey, _Mapping]] = ..., oidc: _Optional[_Union[SignWithOIDC, _Mapping]] = ...) -> None: ...

class SignWithKey(_message.Message):
    __slots__ = ("private_key", "password")
    PRIVATE_KEY_FIELD_NUMBER: _ClassVar[int]
    PASSWORD_FIELD_NUMBER: _ClassVar[int]
    private_key: str
    password: bytes
    def __init__(self, private_key: _Optional[str] = ..., password: _Optional[bytes] = ...) -> None: ...

class SignWithOIDC(_message.Message):
    __slots__ = ("id_token", "options")
    ID_TOKEN_FIELD_NUMBER: _ClassVar[int]
    OPTIONS_FIELD_NUMBER: _ClassVar[int]
    id_token: str
    options: SignOptionsOIDC
    def __init__(self, id_token: _Optional[str] = ..., options: _Optional[_Union[SignOptionsOIDC, _Mapping]] = ...) -> None: ...

class SignResponse(_message.Message):
    __slots__ = ("signature",)
    SIGNATURE_FIELD_NUMBER: _ClassVar[int]
    signature: _signature_pb2.Signature
    def __init__(self, signature: _Optional[_Union[_signature_pb2.Signature, _Mapping]] = ...) -> None: ...

class VerifyRequest(_message.Message):
    __slots__ = ("record_ref", "provider")
    RECORD_REF_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_FIELD_NUMBER: _ClassVar[int]
    record_ref: _record_pb2.RecordRef
    provider: VerifyRequestProvider
    def __init__(self, record_ref: _Optional[_Union[_record_pb2.RecordRef, _Mapping]] = ..., provider: _Optional[_Union[VerifyRequestProvider, _Mapping]] = ...) -> None: ...

class VerifyRequestProvider(_message.Message):
    __slots__ = ("key", "oidc", "any")
    KEY_FIELD_NUMBER: _ClassVar[int]
    OIDC_FIELD_NUMBER: _ClassVar[int]
    ANY_FIELD_NUMBER: _ClassVar[int]
    key: VerifyWithKey
    oidc: VerifyWithOIDC
    any: VerifyWithAny
    def __init__(self, key: _Optional[_Union[VerifyWithKey, _Mapping]] = ..., oidc: _Optional[_Union[VerifyWithOIDC, _Mapping]] = ..., any: _Optional[_Union[VerifyWithAny, _Mapping]] = ...) -> None: ...

class VerifyWithKey(_message.Message):
    __slots__ = ("public_key",)
    PUBLIC_KEY_FIELD_NUMBER: _ClassVar[int]
    public_key: str
    def __init__(self, public_key: _Optional[str] = ...) -> None: ...

class VerifyWithOIDC(_message.Message):
    __slots__ = ("issuer", "subject", "options")
    ISSUER_FIELD_NUMBER: _ClassVar[int]
    SUBJECT_FIELD_NUMBER: _ClassVar[int]
    OPTIONS_FIELD_NUMBER: _ClassVar[int]
    issuer: str
    subject: str
    options: VerifyOptionsOIDC
    def __init__(self, issuer: _Optional[str] = ..., subject: _Optional[str] = ..., options: _Optional[_Union[VerifyOptionsOIDC, _Mapping]] = ...) -> None: ...

class VerifyWithAny(_message.Message):
    __slots__ = ("oidc_options",)
    OIDC_OPTIONS_FIELD_NUMBER: _ClassVar[int]
    oidc_options: VerifyOptionsOIDC
    def __init__(self, oidc_options: _Optional[_Union[VerifyOptionsOIDC, _Mapping]] = ...) -> None: ...

class VerifyResponse(_message.Message):
    __slots__ = ("success", "signers", "error_message")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    SIGNERS_FIELD_NUMBER: _ClassVar[int]
    ERROR_MESSAGE_FIELD_NUMBER: _ClassVar[int]
    success: bool
    signers: _containers.RepeatedCompositeFieldContainer[SignerInfo]
    error_message: str
    def __init__(self, success: bool = ..., signers: _Optional[_Iterable[_Union[SignerInfo, _Mapping]]] = ..., error_message: _Optional[str] = ...) -> None: ...

class SignerInfo(_message.Message):
    __slots__ = ("key", "oidc")
    KEY_FIELD_NUMBER: _ClassVar[int]
    OIDC_FIELD_NUMBER: _ClassVar[int]
    key: SignerInfoKey
    oidc: SignerInfoOIDC
    def __init__(self, key: _Optional[_Union[SignerInfoKey, _Mapping]] = ..., oidc: _Optional[_Union[SignerInfoOIDC, _Mapping]] = ...) -> None: ...

class SignerInfoKey(_message.Message):
    __slots__ = ("public_key", "algorithm")
    PUBLIC_KEY_FIELD_NUMBER: _ClassVar[int]
    ALGORITHM_FIELD_NUMBER: _ClassVar[int]
    public_key: str
    algorithm: str
    def __init__(self, public_key: _Optional[str] = ..., algorithm: _Optional[str] = ...) -> None: ...

class SignerInfoOIDC(_message.Message):
    __slots__ = ("issuer", "subject", "certificate_issuer")
    ISSUER_FIELD_NUMBER: _ClassVar[int]
    SUBJECT_FIELD_NUMBER: _ClassVar[int]
    CERTIFICATE_ISSUER_FIELD_NUMBER: _ClassVar[int]
    issuer: str
    subject: str
    certificate_issuer: str
    def __init__(self, issuer: _Optional[str] = ..., subject: _Optional[str] = ..., certificate_issuer: _Optional[str] = ...) -> None: ...
