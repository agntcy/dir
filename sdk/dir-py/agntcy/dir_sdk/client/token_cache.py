# Copyright AGNTCY Contributors (https://github.com/agntcy)
# SPDX-License-Identifier: Apache-2.0

from __future__ import annotations

import json
import os
from dataclasses import dataclass
from datetime import UTC, datetime, timedelta
from pathlib import Path
from typing import Any

DEFAULT_TOKEN_CACHE_DIR = "dirctl"
TOKEN_CACHE_FILE = "auth-token.json"
DEFAULT_TOKEN_VALIDITY_DURATION = timedelta(hours=8)
TOKEN_EXPIRY_BUFFER = timedelta(minutes=5)
CACHE_DIR_PERMS = 0o700
CACHE_FILE_PERMS = 0o600


def _utcnow() -> datetime:
    now = datetime.now(UTC)
    return now.replace(microsecond=(now.microsecond // 1000) * 1000) # Truncate to milliseconds (3 decimal places) to match other languages

def _parse_timestamp(value: str | None) -> datetime | None:
    if not value:
        return None
    normalized = value.replace("Z", "+00:00")
    
    dt = datetime.fromisoformat(normalized).astimezone(UTC)
    return dt.replace(microsecond=(dt.microsecond // 1000) * 1000) # Truncate to milliseconds (3 decimal places) to match other languages


def _format_timestamp(value: datetime | None) -> str | None:
    if value is None:
        return None
    return value.astimezone(UTC).isoformat(timespec="milliseconds").replace("+00:00", "Z")


@dataclass
class CachedToken:
    access_token: str
    token_type: str = ""
    provider: str = ""
    issuer: str = ""
    refresh_token: str = ""
    expires_at: datetime | None = None
    user: str = ""
    user_id: str = ""
    email: str = ""
    created_at: datetime | None = None

    @classmethod
    def from_json(cls, payload: dict[str, Any]) -> "CachedToken":
        return cls(
            access_token=str(payload.get("access_token", "")),
            token_type=str(payload.get("token_type", "")),
            provider=str(payload.get("provider", "")),
            issuer=str(payload.get("issuer", "")),
            refresh_token=str(payload.get("refresh_token", "")),
            expires_at=_parse_timestamp(payload.get("expires_at")),
            user=str(payload.get("user", "")),
            user_id=str(payload.get("user_id", "")),
            email=str(payload.get("email", "")),
            created_at=_parse_timestamp(payload.get("created_at")),
        )

    def to_json(self) -> dict[str, Any]:
        payload: dict[str, Any] = {
            "access_token": self.access_token,
            "created_at": _format_timestamp(self.created_at or _utcnow()),
        }
        if self.token_type:
            payload["token_type"] = self.token_type
        if self.provider:
            payload["provider"] = self.provider
        if self.issuer:
            payload["issuer"] = self.issuer
        if self.refresh_token:
            payload["refresh_token"] = self.refresh_token
        if self.expires_at is not None:
            payload["expires_at"] = _format_timestamp(self.expires_at)
        if self.user:
            payload["user"] = self.user
        if self.user_id:
            payload["user_id"] = self.user_id
        if self.email:
            payload["email"] = self.email
        return payload


class TokenCache:
    def __init__(self, cache_dir: str | Path | None = None) -> None:
        if cache_dir is None:
            config_home = os.environ.get("XDG_CONFIG_HOME")
            if config_home:
                base_dir = Path(config_home)
            else:
                base_dir = Path.home() / ".config"
            self.cache_dir = base_dir / DEFAULT_TOKEN_CACHE_DIR
        else:
            self.cache_dir = Path(cache_dir)

    def get_cache_path(self) -> Path:
        return self.cache_dir / TOKEN_CACHE_FILE

    def load(self) -> CachedToken | None:
        path = self.get_cache_path()
        if not path.exists():
            return None
        with path.open(encoding="utf-8") as f:
            payload = json.load(f)
        return CachedToken.from_json(payload)

    def save(self, token: CachedToken) -> None:
        self.cache_dir.mkdir(mode=CACHE_DIR_PERMS, parents=True, exist_ok=True)
        os.chmod(self.cache_dir, CACHE_DIR_PERMS)
        path = self.get_cache_path()
        if token.created_at is None:
            token.created_at = _utcnow()
        serialized = json.dumps(token.to_json(), indent=2)
        path.write_text(serialized + "\n", encoding="utf-8")
        os.chmod(path, CACHE_FILE_PERMS)

    def clear(self) -> None:
        path = self.get_cache_path()
        if path.exists():
            path.unlink()

    def is_valid(self, token: CachedToken | None) -> bool:
        if token is None or not token.access_token:
            return False
        now = _utcnow()
        if token.expires_at is None:
            created_at = token.created_at or now
            return now < created_at + DEFAULT_TOKEN_VALIDITY_DURATION
        return now + TOKEN_EXPIRY_BUFFER < token.expires_at

    def get_valid_token(self) -> CachedToken | None:
        token = self.load()
        if not self.is_valid(token):
            return None
        return token
