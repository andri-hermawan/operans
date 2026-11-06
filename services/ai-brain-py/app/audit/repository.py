"""Catatan audit: siapa melakukan apa pada dokumen. Hanya ditambah (akun layanan tidak bisa mengubah/menghapus)."""
import json
from uuid import UUID

from app.shared.database import Database


class AuditRepository:
    def __init__(self, db: Database) -> None:
        self._db = db

    def record(self, *, project_id: UUID | None, user_id: UUID, action: str, document_id: UUID | None = None,
               detail: dict | None = None, request_id: str | None = None) -> None:
        with self._db.pool.connection() as conn:
            conn.execute(
                "INSERT INTO ai.audit_logs (project_id, user_id, action, document_id, detail, request_id)"
                " VALUES (%s, %s, %s, %s, %s::jsonb, %s)",
                (project_id, user_id, action, document_id, json.dumps(detail or {}), request_id),
            )
