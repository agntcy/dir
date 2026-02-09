from agntcy.dir.core.v1 import record_pb2 as _record_pb2
from agntcy.dir.sign.v1 import signature_pb2 as _signature_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Iterable as _Iterable, Mapping as _Mapping, Optional as _Optional, Union as _Union

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
    __slots__ = ("record_ref", "options")
    RECORD_REF_FIELD_NUMBER: _ClassVar[int]
    OPTIONS_FIELD_NUMBER: _ClassVar[int]
    record_ref: _record_pb2.RecordRef
    options: VerifyOptions
    def __init__(self, record_ref: _Optional[_Union[_record_pb2.RecordRef, _Mapping]] = ..., options: _Optional[_Union[VerifyOptions, _Mapping]] = ...) -> None: ...

class VerifyOptions(_message.Message):
    __slots__ = ("key", "oidc")
    KEY_FIELD_NUMBER: _ClassVar[int]
    OIDC_FIELD_NUMBER: _ClassVar[int]
    key: VerifyWithPublicKey
    oidc: VerifyWithOIDCIdentity
    def __init__(self, key: _Optional[_Union[VerifyWithPublicKey, _Mapping]] = ..., oidc: _Optional[_Union[VerifyWithOIDCIdentity, _Mapping]] = ...) -> None: ...

class VerifyWithPublicKey(_message.Message):
    __slots__ = ("public_key",)
    PUBLIC_KEY_FIELD_NUMBER: _ClassVar[int]
    public_key: str
    def __init__(self, public_key: _Optional[str] = ...) -> None: ...

class VerifyWithOIDCIdentity(_message.Message):
    __slots__ = ("issuer", "identity", "trust_root")
    ISSUER_FIELD_NUMBER: _ClassVar[int]
    IDENTITY_FIELD_NUMBER: _ClassVar[int]
    TRUST_ROOT_FIELD_NUMBER: _ClassVar[int]
    issuer: str
    identity: str
    trust_root: TrustRoot
    def __init__(self, issuer: _Optional[str] = ..., identity: _Optional[str] = ..., trust_root: _Optional[_Union[TrustRoot, _Mapping]] = ...) -> None: ...

class TrustRoot(_message.Message):
    __slots__ = ("fulcio_root_pem", "rekor_public_key_pem", "timestamp_authority_roots_pem", "ct_log_public_keys_pem")
    FULCIO_ROOT_PEM_FIELD_NUMBER: _ClassVar[int]
    REKOR_PUBLIC_KEY_PEM_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_AUTHORITY_ROOTS_PEM_FIELD_NUMBER: _ClassVar[int]
    CT_LOG_PUBLIC_KEYS_PEM_FIELD_NUMBER: _ClassVar[int]
    fulcio_root_pem: str
    rekor_public_key_pem: str
    timestamp_authority_roots_pem: _containers.RepeatedScalarFieldContainer[str]
    ct_log_public_keys_pem: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, fulcio_root_pem: _Optional[str] = ..., rekor_public_key_pem: _Optional[str] = ..., timestamp_authority_roots_pem: _Optional[_Iterable[str]] = ..., ct_log_public_keys_pem: _Optional[_Iterable[str]] = ...) -> None: ...

class VerifyResponse(_message.Message):
    __slots__ = ("success", "error_message", "signer_metadata", "signers")
    class SignerMetadataEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    ERROR_MESSAGE_FIELD_NUMBER: _ClassVar[int]
    SIGNER_METADATA_FIELD_NUMBER: _ClassVar[int]
    SIGNERS_FIELD_NUMBER: _ClassVar[int]
    success: bool
    error_message: str
    signer_metadata: _containers.ScalarMap[str, str]
    signers: _containers.RepeatedCompositeFieldContainer[SignerInfo]
    def __init__(self, success: bool = ..., error_message: _Optional[str] = ..., signer_metadata: _Optional[_Mapping[str, str]] = ..., signers: _Optional[_Iterable[_Union[SignerInfo, _Mapping]]] = ...) -> None: ...

class SignerInfo(_message.Message):
    __slots__ = ("key", "oidc")
    KEY_FIELD_NUMBER: _ClassVar[int]
    OIDC_FIELD_NUMBER: _ClassVar[int]
    key: KeySignerInfo
    oidc: OIDCSignerInfo
    def __init__(self, key: _Optional[_Union[KeySignerInfo, _Mapping]] = ..., oidc: _Optional[_Union[OIDCSignerInfo, _Mapping]] = ...) -> None: ...

class KeySignerInfo(_message.Message):
    __slots__ = ("public_key", "algorithm")
    PUBLIC_KEY_FIELD_NUMBER: _ClassVar[int]
    ALGORITHM_FIELD_NUMBER: _ClassVar[int]
    public_key: str
    algorithm: str
    def __init__(self, public_key: _Optional[str] = ..., algorithm: _Optional[str] = ...) -> None: ...

class OIDCSignerInfo(_message.Message):
    __slots__ = ("issuer", "identity")
    ISSUER_FIELD_NUMBER: _ClassVar[int]
    IDENTITY_FIELD_NUMBER: _ClassVar[int]
    issuer: str
    identity: str
    def __init__(self, issuer: _Optional[str] = ..., identity: _Optional[str] = ...) -> None: ...
