"""Koneksi database PostgreSQL (kumpulan koneksi). Layanan tetap menyala walau database belum bisa dihubungi;
/health yang melaporkannya."""
from fastapi import Request
from psycopg_pool import ConnectionPool


class Database:
    def __init__(self, url: str) -> None:
        self.pool = ConnectionPool(url, min_size=1, max_size=5, timeout=5, open=False, name="ai-brain")

    def open(self) -> None:
        self.pool.open(wait=False)  # tidak menunggu: jangan gagal start hanya karena database sedang mati

    def close(self) -> None:
        self.pool.close()


def get_db(request: Request) -> Database:
    """Dependency FastAPI: database yang dirakit di main.create_app."""
    return request.app.state.db
