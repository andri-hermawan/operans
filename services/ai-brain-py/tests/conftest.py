"""Bahan bersama tes: kunci RSA sementara untuk menandatangani token uji, dan database palsu."""
import time
import uuid

import jwt
import pytest
from cryptography.hazmat.primitives import serialization
from cryptography.hazmat.primitives.asymmetric import rsa
from fastapi.testclient import TestClient

from app.main import create_app
from app.shared.config import Settings

ISSUER, AUDIENCE = "simha-core", "simha-api"
PROJECT_ID = str(uuid.uuid4())
USER_ID = str(uuid.uuid4())


class FakeDatabase:
    """Pengganti Database: tidak membuka koneksi apa pun."""

    def open(self) -> None: ...
    def close(self) -> None: ...


@pytest.fixture(scope="session")
def private_key():
    return rsa.generate_private_key(public_exponent=65537, key_size=2048)


@pytest.fixture(scope="session")
def public_key_file(tmp_path_factory, private_key):
    path = tmp_path_factory.mktemp("keys") / "core_public.pem"
    path.write_bytes(
        private_key.public_key().public_bytes(
            serialization.Encoding.PEM, serialization.PublicFormat.SubjectPublicKeyInfo
        )
    )
    return str(path)


@pytest.fixture
def client(public_key_file):
    settings = Settings(
        database_url="postgresql://x:y@localhost/none", jwt_public_key_file=public_key_file,
        jwt_issuer=ISSUER, jwt_audience=AUDIENCE,
    )
    app = create_app(settings, FakeDatabase())
    with TestClient(app) as c:
        yield c


@pytest.fixture
def make_token(private_key):
    """Buat token uji; klaim bisa diubah lewat keyword (mis. exp=..., aud=...)."""

    def _make(key=None, alg="RS256", **override) -> str:
        now = int(time.time())
        claims = {
            "typ": "access", "sub": USER_ID, "name": "Uji", "pid": PROJECT_ID, "prole": "viewer", "sa": False,
            "iss": ISSUER, "aud": AUDIENCE, "iat": now, "exp": now + 900,
        }
        claims.update(override)
        claims = {k: v for k, v in claims.items() if v is not None}
        return jwt.encode(claims, key or private_key, algorithm=alg)

    return _make
