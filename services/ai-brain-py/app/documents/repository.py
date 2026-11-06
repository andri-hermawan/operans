"""Query SQL fitur documents (ai.documents, ai.document_roles, ai.document_pages).

Aturan akses (siapa boleh melihat dokumen apa) ada di access.py dan dijalankan di SQL.
"""
from uuid import UUID

from psycopg.errors import UniqueViolation
from psycopg.rows import dict_row

from app.documents.access import VISIBLE_DOCUMENTS
from app.documents.domain import Document, NewDocument
from app.shared.database import Database
from app.shared.errors import Conflict

_SELECT = """
    SELECT d.id, d.department_id, dp.department_code, dp.department_name, d.title,
           d.document_number, d.revision, d.file_name, d.file_size_bytes, d.page_count,
           d.processing_status, d.processing_error, d.created_at,
           (SELECT coalesce(array_agg(r.role ORDER BY r.role), '{}')
              FROM ai.document_roles r WHERE r.document_id = d.id) AS roles
      FROM ai.documents d
      JOIN ai.departments dp ON dp.id = d.department_id
"""

_WHERE_VISIBLE = " WHERE " + VISIBLE_DOCUMENTS


class DocumentRepository:
    def __init__(self, db: Database) -> None:
        self._db = db

    def department_exists(self, project_id: UUID, department_id: UUID) -> bool:
        with self._db.pool.connection() as conn:
            row = conn.execute(
                "SELECT 1 FROM ai.departments WHERE id = %s AND project_id = %s AND is_active",
                (department_id, project_id),
            ).fetchone()
        return row is not None

    def create(self, *, doc_id: UUID, project_id: UUID, uploader: UUID, data: NewDocument, file_name: str,
               sha256: str, size: int, storage_path: str) -> None:
        """Catat dokumen baru berstatus 'processing' beserta role pembacanya (satu transaksi)."""
        try:
            with self._db.pool.connection() as conn:
                conn.execute(
                    "INSERT INTO ai.documents (id, project_id, department_id, title, document_number,"
                    " revision, file_name, file_sha256, file_size_bytes, storage_path, processing_status, uploaded_by)"
                    " VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, 'processing', %s)",
                    (doc_id, project_id, data.department_id, data.title, data.document_number,
                     data.revision, file_name, sha256, size, storage_path, uploader),
                )
                for role in sorted(set(data.roles)):
                    conn.execute("INSERT INTO ai.document_roles (document_id, role) VALUES (%s, %s)", (doc_id, role))
        except UniqueViolation:
            raise Conflict("file yang sama sudah pernah diunggah di project ini") from None

    def finish(self, doc_id: UUID, pages: list[str]) -> None:
        """Simpan teks per halaman dan tandai 'ready' (satu transaksi)."""
        with self._db.pool.connection() as conn:
            with conn.cursor() as cur:
                cur.executemany(
                    "INSERT INTO ai.document_pages (document_id, page_number, content) VALUES (%s, %s, %s)",
                    [(doc_id, i + 1, text) for i, text in enumerate(pages)],
                )
            conn.execute(
                "UPDATE ai.documents SET processing_status = 'ready', page_count = %s, processing_error = NULL"
                " WHERE id = %s",
                (len(pages), doc_id),
            )

    def reset_for_reprocess(self, doc_id: UUID) -> None:
        """Kosongkan hasil lama dan tandai 'processing' (satu transaksi), sebelum file diproses ulang."""
        with self._db.pool.connection() as conn:
            conn.execute("DELETE FROM ai.document_pages WHERE document_id = %s", (doc_id,))
            conn.execute(
                "UPDATE ai.documents SET processing_status = 'processing', processing_error = NULL, page_count = NULL"
                " WHERE id = %s",
                (doc_id,),
            )

    def fail(self, doc_id: UUID, message: str) -> None:
        with self._db.pool.connection() as conn:
            conn.execute(
                "UPDATE ai.documents SET processing_status = 'failed', processing_error = %s WHERE id = %s",
                (message[:500], doc_id),
            )

    def find_visible(self, *, project_id: UUID, is_admin: bool, role: str | None,
                     department_id: UUID | None) -> list[Document]:
        sql = _SELECT + _WHERE_VISIBLE + """
               AND (%(department_id)s::uuid IS NULL OR d.department_id = %(department_id)s::uuid)
             ORDER BY dp.department_name, d.title"""
        params = {"project_id": project_id, "is_admin": is_admin, "role": role,
                  "department_id": department_id}
        with self._db.pool.connection() as conn, conn.cursor(row_factory=dict_row) as cur:
            cur.execute(sql, params)
            return [Document(**r) for r in cur.fetchall()]

    def get(self, *, project_id: UUID, doc_id: UUID, is_admin: bool, role: str | None) -> Document | None:
        sql = _SELECT + _WHERE_VISIBLE + " AND d.id = %(doc_id)s"
        params = {"project_id": project_id, "is_admin": is_admin, "role": role, "doc_id": doc_id}
        with self._db.pool.connection() as conn, conn.cursor(row_factory=dict_row) as cur:
            cur.execute(sql, params)
            row = cur.fetchone()
        return Document(**row) if row else None

    def file_info(self, doc_id: UUID) -> tuple[str, str] | None:
        """(lokasi file di penyimpanan, nama file asli) — dipakai untuk mencetak."""
        with self._db.pool.connection() as conn:
            row = conn.execute("SELECT storage_path, file_name FROM ai.documents WHERE id = %s", (doc_id,)).fetchone()
        return (row[0], row[1]) if row else None

    def get_page(self, doc_id: UUID, page_number: int) -> str | None:
        with self._db.pool.connection() as conn:
            row = conn.execute(
                "SELECT content FROM ai.document_pages WHERE document_id = %s AND page_number = %s",
                (doc_id, page_number),
            ).fetchone()
        return row[0] if row else None

    def set_roles(self, doc_id: UUID, roles: list[str]) -> None:
        with self._db.pool.connection() as conn:
            conn.execute("DELETE FROM ai.document_roles WHERE document_id = %s", (doc_id,))
            for role in sorted(set(roles)):
                conn.execute("INSERT INTO ai.document_roles (document_id, role) VALUES (%s, %s)", (doc_id, role))

    def deactivate(self, doc_id: UUID) -> None:
        with self._db.pool.connection() as conn:
            conn.execute("UPDATE ai.documents SET is_active = false WHERE id = %s", (doc_id,))
