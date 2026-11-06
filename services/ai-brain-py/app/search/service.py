"""Logika fitur search: validasi kata kunci, batasi jumlah hasil, catat ke audit.
Hak akses mengikuti dokumen (project + role) dan dijalankan di SQL oleh repository."""
from uuid import UUID

from app.audit.repository import AuditRepository
from app.search.domain import SearchResult
from app.search.repository import SearchRepository
from app.shared.errors import Invalid
from app.shared.security import Principal

MIN_QUERY_CHARS = 2
MAX_QUERY_CHARS = 200
DEFAULT_LIMIT = 20
MAX_LIMIT = 50


class SearchService:
    def __init__(self, repo: SearchRepository, audit: AuditRepository) -> None:
        self._repo = repo
        self._audit = audit

    def search(self, p: Principal, query: str, department_id: UUID | None,
               limit: int, request_id: str) -> SearchResult:
        project_id = p.require_project()
        query = query.strip()
        if len(query) < MIN_QUERY_CHARS:
            raise Invalid(f"kata kunci minimal {MIN_QUERY_CHARS} huruf")
        if len(query) > MAX_QUERY_CHARS:
            raise Invalid(f"kata kunci maksimal {MAX_QUERY_CHARS} huruf")
        if not 1 <= limit <= MAX_LIMIT:
            raise Invalid(f"limit harus 1 sampai {MAX_LIMIT}")

        hits = self._repo.search(project_id=project_id, is_admin=p.is_admin, role=p.role, query=query,
                                 department_id=department_id, limit=limit)
        self._audit.record(project_id=project_id, user_id=p.user_id, action="search",
                           detail={"q": query, "hits": len(hits)}, request_id=request_id)
        return SearchResult(query=query, count=len(hits), hits=hits)
