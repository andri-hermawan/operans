"""Query SQL fitur search: pencarian kata kunci (full-text PostgreSQL, bahasa Indonesia) pada teks per halaman.

- Kueri dibaca `websearch_to_tsquery`: aman untuk teks bebas; mendukung "frasa persis", `atau`, dan -kata.
- Ekspresi `to_tsvector('indonesian', content)` HARUS sama dengan indeks migration 000002.
- Hanya dokumen yang boleh dilihat pemanggil (aturan di app/documents/access.py).
"""
from uuid import UUID

from psycopg.rows import dict_row

from app.documents.access import VISIBLE_DOCUMENTS
from app.search.domain import MARK_END, MARK_START, SearchHit
from app.shared.database import Database

_SQL = f"""
    WITH q AS (SELECT websearch_to_tsquery('indonesian', %(q)s) AS tsq)
    SELECT d.id AS document_id, d.title, d.document_number,
           dp.department_code, dp.department_name, pg.page_number,
           ts_headline('indonesian', pg.content, q.tsq,
                       'StartSel={MARK_START}, StopSel={MARK_END}, MaxFragments=1, MinWords=12, MaxWords=35') AS snippet,
           ts_rank_cd(to_tsvector('indonesian', pg.content), q.tsq)::float8 AS score
      FROM q, ai.document_pages pg
      JOIN ai.documents d    ON d.id = pg.document_id
      JOIN ai.departments dp ON dp.id = d.department_id
     WHERE {VISIBLE_DOCUMENTS}
       AND to_tsvector('indonesian', pg.content) @@ q.tsq
       AND (%(department_id)s::uuid IS NULL OR d.department_id = %(department_id)s::uuid)
     ORDER BY score DESC, d.title, pg.page_number
     LIMIT %(limit)s
"""


class SearchRepository:
    def __init__(self, db: Database) -> None:
        self._db = db

    def search(self, *, project_id: UUID, is_admin: bool, role: str | None, query: str,
               department_id: UUID | None, limit: int) -> list[SearchHit]:
        params = {"project_id": project_id, "is_admin": is_admin, "role": role, "q": query,
                  "department_id": department_id, "limit": limit}
        with self._db.pool.connection() as conn, conn.cursor(row_factory=dict_row) as cur:
            cur.execute(_SQL, params)
            return [SearchHit(**r) for r in cur.fetchall()]
