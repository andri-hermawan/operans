"""Tes login: token buatan core-api diterima; semua bentuk token salah ditolak dengan 401."""
import time

from cryptography.hazmat.primitives.asymmetric import rsa

from tests.conftest import PROJECT_ID, USER_ID


def auth(token: str) -> dict:
    return {"Authorization": f"Bearer {token}"}


def test_token_valid_diterima(client, make_token):
    r = client.get("/api/v1/me", headers=auth(make_token()))
    assert r.status_code == 200
    body = r.json()
    assert body["user_id"] == USER_ID and body["project_id"] == PROJECT_ID
    assert body["role"] == "viewer" and body["is_superadmin"] is False


def test_tanpa_token_ditolak(client):
    r = client.get("/api/v1/me")
    assert r.status_code == 401
    assert "token" in r.json()["error"]


def test_token_kedaluwarsa_ditolak(client, make_token):
    r = client.get("/api/v1/me", headers=auth(make_token(exp=int(time.time()) - 3600)))
    assert r.status_code == 401


def test_tanpa_exp_ditolak(client, make_token):
    r = client.get("/api/v1/me", headers=auth(make_token(exp=None)))
    assert r.status_code == 401


def test_audience_atau_issuer_salah_ditolak(client, make_token):
    assert client.get("/api/v1/me", headers=auth(make_token(aud="lain"))).status_code == 401
    assert client.get("/api/v1/me", headers=auth(make_token(iss="lain"))).status_code == 401


def test_ditandatangani_kunci_lain_ditolak(client, make_token):
    palsu = rsa.generate_private_key(public_exponent=65537, key_size=2048)
    assert client.get("/api/v1/me", headers=auth(make_token(key=palsu))).status_code == 401


def test_bukan_access_token_ditolak(client, make_token):
    assert client.get("/api/v1/me", headers=auth(make_token(typ="refresh"))).status_code == 401


def test_algoritma_hs256_ditolak(client, make_token):
    # penyerang menandatangani HS256 memakai kunci publik sebagai "rahasia" — harus tetap ditolak
    token = make_token(key="rahasia-apa-saja-yang-cukup-panjang-32b", alg="HS256")
    assert client.get("/api/v1/me", headers=auth(token)).status_code == 401


def test_user_biasa_tanpa_project_ditolak(client, make_token):
    assert client.get("/api/v1/me", headers=auth(make_token(pid=None))).status_code == 401


def test_superadmin_tanpa_project_diterima(client, make_token):
    r = client.get("/api/v1/me", headers=auth(make_token(sa=True, pid=None, prole=None)))
    assert r.status_code == 200
    assert r.json()["project_id"] is None and r.json()["is_superadmin"] is True
