"""Query SQL fitur health. Hanya membaca katalog PostgreSQL."""
from app.shared.database import Database


class HealthRepository:
    def __init__(self, db: Database) -> None:
        self._db = db

    def ping(self) -> None:
        with self._db.pool.connection(timeout=2) as conn:
            conn.execute("SELECT 1")

    def schema_ai_exists(self) -> bool:
        with self._db.pool.connection(timeout=2) as conn:
            row = conn.execute("SELECT EXISTS (SELECT 1 FROM pg_namespace WHERE nspname = 'ai')").fetchone()
        return bool(row and row[0])

    def pgvector_version(self) -> str | None:
        with self._db.pool.connection(timeout=2) as conn:
            row = conn.execute("SELECT extversion FROM pg_extension WHERE extname = 'vector'").fetchone()
        return row[0] if row else None
