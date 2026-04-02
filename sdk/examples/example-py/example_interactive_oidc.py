# Copyright AGNTCY Contributors (https://github.com/agntcy)
# SPDX-License-Identifier: Apache-2.0

from __future__ import annotations

import argparse
import os

from agntcy.dir_sdk.client import Client, Config, OAuthPkceError
from agntcy.dir_sdk.models import search_v1

DEFAULT_OIDC_ISSUER = "https://dev.idp.ads.outshift.io"
DEFAULT_SERVER_ADDRESS = "dev.gateway.ads.outshift.io:443"
DEFAULT_TLS_SERVER_NAME = "dev.gateway.ads.outshift.io"
DEFAULT_REDIRECT_URI = "http://localhost:8484/callback"


def require_env(name: str) -> str:
    value = os.environ.get(name, "").strip()
    if not value:
        msg = f"{name} is required for the interactive OIDC example"
        raise RuntimeError(msg)
    return value


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Interactive OIDC example that calls SearchCIDs only.",
    )
    parser.add_argument(
        "--version",
        default="v1*",
        help="Version query used for SearchCIDs (default: v1*)",
    )
    parser.add_argument(
        "--limit",
        type=int,
        default=3,
        help="Maximum number of CIDs to return (default: 3)",
    )
    return parser.parse_args()


def build_client() -> Client:
    config = Config(
        server_address=os.environ.get(
            "DIRECTORY_CLIENT_SERVER_ADDRESS",
            DEFAULT_SERVER_ADDRESS,
        ),
        auth_mode="oidc",
        oidc_issuer=DEFAULT_OIDC_ISSUER,
        oidc_client_id=require_env("DIRECTORY_CLIENT_OIDC_CLIENT_ID"),
        oidc_client_secret=os.environ.get("DIRECTORY_CLIENT_OIDC_CLIENT_SECRET", ""),
        tls_server_name=os.environ.get(
            "DIRECTORY_CLIENT_TLS_SERVER_NAME",
            DEFAULT_TLS_SERVER_NAME,
        ),
        oidc_redirect_uri=os.environ.get(
            "DIRECTORY_CLIENT_OIDC_REDIRECT_URI",
            DEFAULT_REDIRECT_URI,
        ),
        oidc_callback_port=int(
            os.environ.get("DIRECTORY_CLIENT_OIDC_CALLBACK_PORT", "8484"),
        ),
    )
    client = Client(config)
    holder = getattr(client, "_oauth_holder", None)
    if holder is not None:
        try:
            holder.get_access_token()
            print("Using cached OIDC token.")
            return client
        except RuntimeError:
            pass

    print("No cached OIDC token found. Starting interactive login.")
    client.authenticate_oauth_pkce()
    return client


def main() -> None:
    args = parse_args()
    client = build_client()

    search_query = search_v1.RecordQuery(
        type=search_v1.RECORD_QUERY_TYPE_VERSION,
        value=args.version,
    )
    search_request = search_v1.SearchCIDsRequest(
        queries=[search_query],
        limit=args.limit,
    )
    objects = list(client.search_cids(search_request))
    print(f"SearchCIDs results for version {args.version!r}:")
    for obj in objects:
        print(obj)


if __name__ == "__main__":
    try:
        main()
    except OAuthPkceError as e:
        print(f"Interactive OIDC login failed: {e}")
        raise
