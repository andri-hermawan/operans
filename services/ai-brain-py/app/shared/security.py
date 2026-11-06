"""Verifikasi access token buatan core-api (RS256) — padanan internal/shared/middleware/jwt.go di service Go.
AI Brain hanya MEMERIKSA token; tidak menerbitkannya."""
from dataclasses import dataclass
from pathlib import Path
from uuid import UUID

import jwt
from cryptography.hazmat.primitives import serialization
from fastapi import Depends, HTTPException, Request
from fastapi.security import HTTPAuthorizationCredentials, HTTPBearer

from app.shared.errors import Invalid

ROLES = {"admin", "dispatcher", "viewer"}


@dataclass(frozen=True)
class Principal:
    """SIAPA yang sedang memanggil API, diambil dari token yang sudah diverifikasi."""

    user_id: UUID
    name: str
    project_id: UUID | None
    role: str | None  # role di project aktif (None untuk superadmin tanpa project)
    is_superadmin: bool

    @property
    def project_scope(self) -> UUID | None:
        """Project yang boleh dilihat. None = semua project (hanya superadmin yang belum memilih project)."""
        return None if self.is_superadmin and self.project_id is None else self.project_id

    @property
    def is_admin(self) -> bool:
        """Boleh mengelola dokumen: superadmin atau admin di project aktifnya."""
        return self.is_superadmin or self.role == "admin"

    def require_project(self) -> UUID:
        """Project aktif yang wajib ada untuk bekerja dengan dokumen (superadmin harus memilih project dulu)."""
        if self.project_id is None:
            raise Invalid("pilih project terlebih dahulu")
        return self.project_id


class TokenVerifier:
    def __init__(self, public_key_pem: bytes, issuer: str, audience: str) -> None:
        self._key = serialization.load_pem_public_key(public_key_pem)  # gagal di awal bila kunci rusak
        self._issuer = issuer
        self._audience = audience

    @classmethod
    def from_file(cls, path: str, issuer: str, audience: str) -> "TokenVerifier":
        return cls(Path(path).read_bytes(), issuer, audience)

    def verify(self, raw: str) -> Principal:
        try:
            claims = jwt.decode(
                raw,
                self._key,
                algorithms=["RS256"],  # tolak HS256, "none", dll.
                issuer=self._issuer,  # hanya token dari core-api
                audience=self._audience,  # hanya token yang memang untuk API SIMHA
                leeway=30,  # toleransi selisih jam antar-server
                options={"require": ["exp", "sub"]},  # token tanpa exp ditolak
            )
            return _principal_from(claims)
        except (jwt.PyJWTError, ValueError) as err:
            raise InvalidToken from err


class InvalidToken(Exception):
    pass


def _principal_from(c: dict) -> Principal:
    if c.get("typ") != "access":
        raise ValueError("bukan access token")
    is_sa = bool(c.get("sa"))
    project_id = UUID(c["pid"]) if c.get("pid") else None
    role = c.get("prole") or None
    if not is_sa and (project_id is None or role not in ROLES):
        raise ValueError("klaim pid/prole wajib untuk user biasa")
    return Principal(user_id=UUID(c["sub"]), name=c.get("name", ""), project_id=project_id, role=role, is_superadmin=is_sa)


_bearer = HTTPBearer(auto_error=False)


def current_principal(
    request: Request, cred: HTTPAuthorizationCredentials | None = Depends(_bearer)
) -> Principal:
    """Dependency FastAPI: wajib login. Dipakai semua endpoint kecuali /health."""
    if cred is None:
        raise HTTPException(401, "token diperlukan (Authorization: Bearer <token>)")
    try:
        return request.app.state.verifier.verify(cred.credentials)
    except InvalidToken:
        raise HTTPException(401, "token tidak valid atau kedaluwarsa") from None
