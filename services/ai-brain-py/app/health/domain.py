"""Bentuk data fitur health (hanya struktur, tanpa logika)."""
from pydantic import BaseModel


class HealthReport(BaseModel):
    status: str  # "ok" | "degraded"
    service: str = "ai-brain"
    database: bool  # database bisa dihubungi
    schema_ai: bool  # schema `ai` sudah dibuat (migration jalan)
    pgvector: str | None  # versi ekstensi pgvector; None = belum aktif (boleh, ditunda)
