"""Konfigurasi layanan: dibaca dari variabel lingkungan / file .env (padanan internal/shared/config di service Go)."""
from functools import lru_cache

from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    model_config = SettingsConfigDict(env_file=".env", extra="ignore")

    database_url: str  # wajib; format psycopg, tanpa ?schema=public
    jwt_public_key_file: str  # kunci PUBLIK RS256 milik core-api
    jwt_issuer: str = "simha-core"
    jwt_audience: str = "simha-api"
    log_level: str = "INFO"
    storage_dir: str = "storage"  # tempat file PDF (di luar Git; bukan folder yang dilayani web)
    max_upload_mb: int = 25
    ocr_enabled: bool = False  # OCR halaman scan; butuh `pip install -e ".[ocr]"`


@lru_cache
def get_settings() -> Settings:
    return Settings()
