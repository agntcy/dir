#!/usr/bin/env python3
# Copyright AGNTCY Contributors (https://github.com/agntcy)
# SPDX-License-Identifier: Apache-2.0
"""Live Directory/AGNTCY E2E: real Node, OIDC enrollment, OCI and signed claims.

Requires Docker and the three Linux binaries documented in README.md.
Containers and state are isolated; cleanup always runs. No service is mocked.
"""

import argparse
import hashlib
import http.server
import json
import os
from pathlib import Path
import socket
import ssl
import subprocess
import tempfile
import threading
import time
import urllib.error
import urllib.request
from datetime import datetime, timedelta, timezone

import jwt
from cryptography import x509
from cryptography.hazmat.primitives import hashes, serialization
from cryptography.hazmat.primitives.asymmetric import rsa
from cryptography.x509.oid import NameOID

HERE = Path(__file__).resolve().parent
ROOT = HERE.parents[2]


def check(condition, description):
    if not condition:
        raise AssertionError(description)
    print("PASS:", description, flush=True)


def docker(*args):
    return subprocess.check_output(["docker", *map(str, args)], text=True).strip()


def request(url, body=None, token=None):
    data = None if body is None else json.dumps(body).encode()
    headers = {"Content-Type": "application/json"}
    if token:
        headers["Authorization"] = "Bearer " + token
    with urllib.request.urlopen(
        urllib.request.Request(url, data, headers), timeout=15
    ) as response:
        return json.load(response)


def eventually(fn, description, seconds=90):
    end = time.monotonic() + seconds
    last = None
    while time.monotonic() < end:
        try:
            if fn():
                return
        except (OSError, ValueError, subprocess.CalledProcessError) as exc:
            last = exc
        time.sleep(1)
    raise AssertionError(f"Timed out: {description}; last error: {last}")


def private_key(path):
    key = rsa.generate_private_key(public_exponent=65537, key_size=2048)
    path.write_bytes(
        key.private_bytes(
            serialization.Encoding.PEM,
            serialization.PrivateFormat.PKCS8,
            serialization.NoEncryption(),
        )
    )
    return key


def certificate(directory):
    key = private_key(directory / "tls.key")
    name = x509.Name([x509.NameAttribute(NameOID.COMMON_NAME, "Directory E2E fixture")])
    now = datetime.now(timezone.utc)
    cert = (
        x509.CertificateBuilder()
        .subject_name(name)
        .issuer_name(name)
        .public_key(key.public_key())
        .serial_number(x509.random_serial_number())
        .not_valid_before(now - timedelta(minutes=1))
        .not_valid_after(now + timedelta(days=1))
        .add_extension(x509.BasicConstraints(ca=True, path_length=None), critical=True)
        .add_extension(
            x509.SubjectAlternativeName(
                [x509.DNSName(n) for n in ("host.docker.internal", "verifier")]
            ),
            critical=False,
        )
        .sign(key, hashes.SHA256())
    )
    (directory / "tls.crt").write_bytes(cert.public_bytes(serialization.Encoding.PEM))


class NodeTLSProxy(http.server.BaseHTTPRequestHandler):
    """TLS termination only; all ID, credential and proof operations reach real Node."""

    node_url = ""

    def log_message(self, *_):
        pass

    def forward(self):
        body = self.rfile.read(int(self.headers.get("Content-Length", 0))) or None
        upstream = urllib.request.Request(
            self.node_url + self.path,
            body,
            {"Content-Type": "application/json"},
            method=self.command,
        )
        try:
            response = urllib.request.urlopen(upstream, timeout=15)
        except urllib.error.HTTPError as exc:
            response = exc
        except OSError:
            self.send_error(503)
            return
        with response:
            data = response.read()
            self.send_response(response.status)
            self.send_header(
                "Content-Type", response.headers.get("Content-Type", "application/json")
            )
            self.send_header("Content-Length", str(len(data)))
            self.end_headers()
            self.wfile.write(data)

    do_GET = forward
    do_POST = forward


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--dirctl", type=Path, required=True)
    parser.add_argument("--verifier", type=Path, required=True)
    parser.add_argument("--node", type=Path, required=True)
    parser.add_argument("--port-base", type=int, default=19300)
    parser.add_argument("--artifacts", type=Path, required=True)
    args = parser.parse_args()
    for binary in (args.dirctl, args.verifier, args.node):
        if not binary.is_file():
            parser.error(f"Missing binary: {binary}")
    args.artifacts.mkdir(parents=True, exist_ok=True)
    (args.artifacts / "results.json").unlink(missing_ok=True)
    prefix = "dir-agntcy-e2e-" + os.urandom(4).hex()
    containers = []
    proxy = None
    ports = {
        k: args.port_base + offset
        for offset, k in enumerate(
            ("keycloak", "node", "proxy", "verifier", "grpc", "oci")
        )
    }
    for port in ports.values():
        with socket.socket() as probe:
            probe.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
            probe.bind(("0.0.0.0", port))
    with tempfile.TemporaryDirectory(prefix=prefix) as tmp:
        work = Path(tmp)
        certificate(work)
        result_key = private_key(work / "result.key")
        (work / "result.pub").write_bytes(
            result_key.public_key().public_bytes(
                serialization.Encoding.PEM,
                serialization.PublicFormat.SubjectPublicKeyInfo,
            )
        )
        (work / "verifier.token").write_text("disposable-verifier-token")
        docker("network", "create", prefix)

        def start(name, image, *options, command=()):
            container = prefix + "-" + name
            containers.append(container)
            docker(
                "run",
                "-d",
                "--name",
                container,
                "--network",
                prefix,
                "--network-alias",
                name,
                "--add-host",
                "host.docker.internal:host-gateway",
                *options,
                image,
                *command,
            )
            return container

        def cli(*arguments):
            result = subprocess.run(
                [
                    "docker",
                    "exec",
                    prefix + "-directory",
                    "/work/dirctl",
                    "--server-addr",
                    "localhost:8888",
                    *map(str, arguments),
                ],
                capture_output=True,
                text=True,
                timeout=25,
            )
            if result.returncode:
                raise subprocess.CalledProcessError(
                    result.returncode,
                    result.args,
                    output=result.stdout,
                    stderr=result.stderr,
                )
            return result.stdout.strip()

        try:
            realm = {
                "realm": "primary",
                "enabled": True,
                "clients": [
                    {
                        "clientId": "primary-client",
                        "secret": "disposable-client-secret",
                        "enabled": True,
                        "publicClient": False,
                        "serviceAccountsEnabled": True,
                        "standardFlowEnabled": False,
                        "protocol": "openid-connect",
                        "protocolMappers": [
                            {
                                "name": "identity-audience",
                                "protocol": "openid-connect",
                                "protocolMapper": "oidc-audience-mapper",
                                "config": {
                                    "included.custom.audience": "identity-node",
                                    "access.token.claim": "true",
                                },
                            }
                        ],
                    }
                ],
            }
            (work / "realm.json").write_text(json.dumps(realm))
            issuer_url = f"http://host.docker.internal:{ports['keycloak']}"
            start(
                "postgres",
                "postgres:16",
                "-e",
                "POSTGRES_DB=identity",
                "-e",
                "POSTGRES_USER=identity",
                "-e",
                "POSTGRES_PASSWORD=disposable-db-password",
            )
            start(
                "keycloak",
                "quay.io/keycloak/keycloak:26.7",
                "-p",
                f"{ports['keycloak']}:8080",
                "-v",
                f"{work}/realm.json:/opt/keycloak/data/import/realm.json:ro",
                "-e",
                "KC_BOOTSTRAP_ADMIN_USERNAME=test",
                "-e",
                "KC_BOOTSTRAP_ADMIN_PASSWORD=disposable-admin-password",
                command=("start-dev", "--import-realm", "--hostname", issuer_url),
            )
            node_url = f"http://127.0.0.1:{ports['node']}"
            eventually(
                lambda: (
                    "accepting connections"
                    in docker(
                        "exec", prefix + "-postgres", "pg_isready", "-U", "identity"
                    )
                ),
                "Postgres startup",
            )
            start(
                "node",
                "golang:1.27.1-bookworm",
                "-v",
                f"{args.node.resolve()}:/node:ro",
                "-p",
                f"{ports['node']}:4000",
                "-e",
                "DB_HOST=postgres",
                "-e",
                "DB_PORT=5432",
                "-e",
                "DB_NAME=identity",
                "-e",
                "DB_USERNAME=identity",
                "-e",
                "DB_PASSWORD=disposable-db-password",
                "-e",
                "DB_USE_SSL=false",
                command=("/node",),
            )
            eventually(
                lambda: request(
                    f"http://127.0.0.1:{ports['keycloak']}/realms/primary/.well-known/openid-configuration"
                ),
                "Keycloak startup",
            )
            NodeTLSProxy.node_url = node_url
            proxy = http.server.ThreadingHTTPServer(
                ("0.0.0.0", ports["proxy"]), NodeTLSProxy
            )
            ctx = ssl.SSLContext(ssl.PROTOCOL_TLS_SERVER)
            ctx.load_cert_chain(work / "tls.crt", work / "tls.key")
            proxy.socket = ctx.wrap_socket(proxy.socket, server_side=True)
            threading.Thread(target=proxy.serve_forever, daemon=True).start()
            token_body = b"grant_type=client_credentials&client_id=primary-client&client_secret=disposable-client-secret"
            with urllib.request.urlopen(
                urllib.request.Request(
                    f"http://127.0.0.1:{ports['keycloak']}/realms/primary/protocol/openid-connect/token",
                    token_body,
                ),
                timeout=15,
            ) as response:
                token = json.load(response)["access_token"]
            # This fixture uses the trusted disposable Keycloak; Node validates the enrollment proof.
            agent_id = (
                "IDP-" + jwt.decode(token, options={"verify_signature": False})["sub"]
            )
            agent_key = private_key(work / "agent.key")
            public = json.loads(
                jwt.algorithms.RSAAlgorithm.to_jwk(agent_key.public_key())
            ) | {"kid": "e2e-signer", "use": "sig", "alg": "RS256"}
            issuer = {
                "commonName": "host.docker.internal",
                "organization": "e2e",
                "publicKey": public,
                "authType": "ISSUER_AUTH_TYPE_IDP",
            }
            proof = {"type": "JWT", "proofValue": token}
            eventually(
                lambda: (
                    request(
                        node_url + "/v1alpha1/issuer/register",
                        {"issuer": issuer, "proof": proof},
                    )
                    is not None
                ),
                "real Node enrollment",
            )
            request(
                node_url + "/v1alpha1/id/generate", {"issuer": issuer, "proof": proof}
            )
            subject = "agntcy://" + agent_id
            check(
                request(node_url + "/v1alpha1/id/resolve", {"id": agent_id})[
                    "resolverMetadata"
                ]["id"]
                == agent_id,
                "real Node resolves enrolled identity",
            )
            start(
                "verifier",
                "golang:1.27.1-bookworm",
                "-v",
                f"{args.verifier.resolve()}:/verifier:ro",
                "-v",
                f"{work}:/fixtures:ro",
                "-p",
                f"{ports['verifier']}:8443",
                "-e",
                f"IDENTITY_NODE_URL=https://host.docker.internal:{ports['proxy']}",
                "-e",
                "SSL_CERT_FILE=/fixtures/tls.crt",
                "-e",
                "RESULT_SIGNING_KEY_PATH=/fixtures/result.key",
                "-e",
                "VERIFIER_BEARER_TOKEN_FILE=/fixtures/verifier.token",
                "-e",
                "TLS_CERT_FILE=/fixtures/tls.crt",
                "-e",
                "TLS_KEY_FILE=/fixtures/tls.key",
                "-e",
                "RESULT_TTL=20s",
                command=("/verifier",),
            )
            config = """server:
  listen_address: ':8888'
  store:
    provider: oci
    oci:
      local_dir: /state/store
      registry_address: 0.0.0.0:5555
      repository_name: dir
      auth_config:
        insecure: true
  routing:
    listen_address: /ip4/127.0.0.1/tcp/0
    key_path: /state/node.key
  database:
    type: sqlite
    sqlite:
      path: /state/dir.db
  extractor:
    asset_dir: /nonexistent-extractor-assets-e2e
  publication:
    scheduler_interval: 1h
    worker_count: 1
    worker_timeout: 30m
reconciler:
  local_registry:
    registry_address: localhost:5555
    repository_name: dir
    auth_config:
      insecure: true
  indexer:
    enabled: true
    interval: 2s
  identity:
    enabled: true
    interval: 2s
    record_timeout: 15s
    agntcy:
      verifier_url: https://verifier:8443
      verifier_trust_bundle_file: /fixtures/result.pub
      bearer_token_file: /fixtures/verifier.token
      require_agent_badge: true
      request_timeout: 10s
      max_verification_age: 20s
"""
            (work / "daemon.yaml").write_text(config)
            start(
                "directory",
                "golang:1.27.1-bookworm",
                "-v",
                f"{args.dirctl.resolve()}:/work/dirctl:ro",
                "-v",
                f"{work}:/fixtures",
                "-p",
                f"{ports['grpc']}:8888",
                "-p",
                f"{ports['oci']}:5555",
                "-e",
                "SSL_CERT_FILE=/fixtures/tls.crt",
                command=(
                    "/work/dirctl",
                    "daemon",
                    "start",
                    "--config",
                    "/fixtures/daemon.yaml",
                    "--data-dir",
                    "/state",
                ),
            )
            eventually(
                lambda: cli("search", "--name", "example.com/agntcy-e2e/*") is not None,
                "Directory daemon startup",
            )

            def push(name, badge=True, seconds=300):
                record = json.loads(
                    (ROOT / "tests/e2e/shared/testdata/record_080_v4.json").read_text()
                )
                record["name"] = "example.com/agntcy-e2e/" + name
                record["annotations"] = {"agntcy.dir/identity": subject}
                path = work / (name + ".json")
                path.write_text(json.dumps(record))
                cid = cli("push", "/fixtures/" + path.name, "--output", "json")
                parsed = json.loads(cid)
                cid = parsed["cid"] if isinstance(parsed, dict) else parsed
                if badge:
                    now = datetime.now(timezone.utc).replace(microsecond=0)
                    context = ["https://www.w3.org/2018/credentials/v1"]
                    vc = {
                        "@context": context,
                        "context": context,
                        "type": ["VerifiableCredential", "AgentBadge"],
                        "id": "urn:uuid:" + os.urandom(16).hex(),
                        "issuer": "host.docker.internal",
                        "issuanceDate": now.isoformat().replace("+00:00", "Z"),
                        "expirationDate": (now + timedelta(seconds=seconds))
                        .isoformat()
                        .replace("+00:00", "Z"),
                        "credentialSubject": {"id": agent_id, "badge": record},
                    }
                    signed = jwt.encode(
                        vc,
                        agent_key,
                        algorithm="RS256",
                        headers={"kid": "e2e-signer", "typ": "JOSE"},
                    )
                    envelope = {
                        "envelopeType": "CREDENTIAL_ENVELOPE_TYPE_JOSE",
                        "value": signed,
                    }
                    check(
                        request(node_url + "/v1alpha1/vc/verify", {"vc": envelope})[
                            "status"
                        ]
                        is True,
                        name + ": real Node accepts badge signature",
                    )
                    request(
                        node_url + "/v1alpha1/vc/publish",
                        {"vc": envelope, "proof": proof},
                    )
                return cid

            def status(cid):
                return json.loads(cli("identity", "status", cid, "--output", "json"))

            def matches(cid, expected):
                row = status(cid).get("identity") or {}
                return row.get("status") == expected

            def claim(cid, key="agent.key"):
                cli(
                    "identity",
                    "claim",
                    "--record",
                    cid,
                    "--role",
                    "identity",
                    "--key",
                    "/fixtures/" + key,
                )

            def search():
                return cli("search", "--identity", subject, "--identity-verified")

            good = push("valid")
            check(
                good not in search(),
                "unclaimed record is excluded from verified identity search",
            )
            claim(good)
            eventually(lambda: matches(good, "verified"), "valid claim verified")
            check(
                good in search(),
                "valid native claim + matching badge appears in verified identity search",
            )
            # Verify the native referrer is in actual OCI storage, with manifest subject binding.
            check(
                bool(cli("pull", good, "--output", "json")),
                "record remains retrievable by content CID",
            )
            registry = f"http://127.0.0.1:{ports['oci']}/v2/dir"
            manifest_request = urllib.request.Request(
                registry + "/manifests/" + good,
                headers={"Accept": "application/vnd.oci.image.manifest.v1+json"},
            )
            with urllib.request.urlopen(manifest_request, timeout=10) as response:
                manifest_digest = (
                    "sha256:" + hashlib.sha256(response.read()).hexdigest()
                )
            index = request(registry + "/referrers/" + manifest_digest)
            claim_manifests = [
                request(registry + "/manifests/" + ref["digest"])
                for ref in index["manifests"]
            ]
            native_claim = next(
                m
                for m in claim_manifests
                if "agntcy.dir.identity.v1.IdentityClaim"
                in m.get("annotations", {}).values()
            )
            check(
                native_claim["subject"]["digest"] == manifest_digest,
                "native OCI IdentityClaim manifest is bound to the exact record manifest",
            )
            claim_blob = request(
                registry + "/blobs/" + native_claim["layers"][0]["digest"]
            )
            check(
                claim_blob["data"]["recordCid"] == good
                and claim_blob["data"]["subject"] == subject,
                "stored claim JSON carries exact CID and subject",
            )
            mismatch = push("no-matching-badge", badge=False)
            claim(mismatch)
            eventually(
                lambda: matches(mismatch, "failed"),
                "other CID rejects shared-agent evidence",
            )
            check(
                mismatch not in search(),
                "same Agent ID cannot reuse another record badge",
            )
            private_key(work / "wrong.key")
            wrong = push("wrong-key")
            claim(wrong, "wrong.key")
            eventually(lambda: matches(wrong, "failed"), "wrong claim key rejected")
            check(
                wrong not in search(),
                "valid badge does not authorize an unrelated claim signer",
            )
            expired = push("expired-badge", seconds=-1)
            claim(expired)
            eventually(lambda: matches(expired, "failed"), "expired badge rejected")
            check(
                expired not in search(),
                "expired badge is excluded from verified identity search",
            )
            # Wait until the successful observation is fresh, then take verifier offline.
            eventually(
                lambda: matches(good, "verified"), "verified record before outage"
            )
            docker("stop", prefix + "-verifier")
            eventually(
                lambda: matches(good, "failed"),
                "signed deadline expires during outage",
                seconds=45,
            )
            check(
                good not in search(),
                "verifier outage cannot extend verified search past the signed deadline",
            )
            docker("start", prefix + "-verifier")
            eventually(lambda: matches(good, "verified"), "verification recovers")
            check(good in search(), "verified search recovers after verifier restarts")
            (args.artifacts / "results.json").write_text(
                json.dumps(
                    {
                        "result": "passed",
                        "agent_subject": subject,
                        "valid_cid": good,
                        "badge_mismatch_cid": mismatch,
                        "wrong_key_cid": wrong,
                        "expired_badge_cid": expired,
                    },
                    indent=2,
                )
            )
        finally:
            for container in containers:
                logs = subprocess.run(
                    ["docker", "logs", container], capture_output=True, text=True
                )
                (args.artifacts / (container.rsplit("-", 1)[-1] + ".log")).write_text(
                    logs.stdout + logs.stderr
                )
            if proxy:
                proxy.shutdown()
                proxy.server_close()
            for container in reversed(containers):
                docker("rm", "-f", container)
            docker("network", "rm", prefix)


if __name__ == "__main__":
    main()
