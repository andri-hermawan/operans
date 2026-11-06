"""Pintu HTTP fitur search: GET /api/v1/search?q=..."""
from uuid import UUID

from fastapi import APIRouter, Depends, Request

from app.audit.repository import AuditRepository
from app.search.domain import SearchResult
from app.search.repository import SearchRepository
from app.search.service import DEFAULT_LIMIT, SearchService
from app.shared.database import Database, get_db
from app.shared.security import Principal, current_principal

router = APIRouter(prefix="/api/v1/search", tags=["search"])


def get_service(db: Database = Depends(get_db)) -> SearchService:
    return SearchService(SearchRepository(db), AuditRepository(db))


@router.get("", response_model=SearchResult)
def search(
    request: Request,
    q: str,
    department_id: UUID | None = None,
    limit: int = DEFAULT_LIMIT,
    p: Principal = Depends(current_principal),
    svc: SearchService = Depends(get_service),
):
    return svc.search(p, q, department_id, limit, request.state.request_id)
