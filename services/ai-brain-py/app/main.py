"""Perakit layanan AI Brain (padanan cmd/api/main.go di service Go): baca konfigurasi, sambungkan database,
pasang pemeriksa token, daftarkan semua fitur. Logika bisnis TIDAK ada di sini.

Jalankan:  uvicorn app.main:build_app --factory --port 3348
"""
from contextlib import asynccontextmanager

from fastapi import FastAPI

from app.departments import router as departments
from app.documents import router as documents
from app.documents.ocr import build_ocr
from app.health import router as health
from app.me import router as me
from app.search import router as search
from app.shared.config import Settings, get_settings
from app.shared.database import Database
from app.shared.observability import install_http_basics, setup_logging
from app.shared.security import TokenVerifier
from app.shared.storage import FileStore


def create_app(settings: Settings | None = None, db: Database | None = None) -> FastAPI:
    """settings & db boleh diganti (dipakai tes); bila kosong dibaca dari .env."""
    settings = settings or get_settings()
    setup_logging(settings.log_level)
    db = db or Database(settings.database_url)

    @asynccontextmanager
    async def lifespan(app: FastAPI):
        db.open()
        yield
        db.close()

    app = FastAPI(title="SIMHA AI Brain", lifespan=lifespan)
    app.state.settings = settings
    app.state.db = db
    app.state.store = FileStore(settings.storage_dir)
    app.state.ocr = build_ocr(settings.ocr_enabled)
    app.state.verifier = TokenVerifier.from_file(settings.jwt_public_key_file, settings.jwt_issuer, settings.jwt_audience)
    install_http_basics(app)

    # Daftar fitur — tambah satu baris per fitur baru (chat, ...)
    app.include_router(health.router)
    app.include_router(me.router)
    app.include_router(departments.router)
    app.include_router(documents.router)
    app.include_router(search.router)
    return app


def build_app() -> FastAPI:
    return create_app()
