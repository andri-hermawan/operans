"""GET /api/v1/me — siapa saya menurut token. Dipakai untuk menguji login lintas layanan (core-api → AI Brain)."""
from uuid import UUID

from fastapi import APIRouter, Depends
from pydantic import BaseModel

from app.shared.security import Principal, current_principal

router = APIRouter(prefix="/api/v1")


class MeResponse(BaseModel):
    user_id: UUID
    name: str
    project_id: UUID | None
    role: str | None
    is_superadmin: bool


@router.get("/me", response_model=MeResponse)
def me(p: Principal = Depends(current_principal)) -> MeResponse:
    return MeResponse(
        user_id=p.user_id, name=p.name, project_id=p.project_id, role=p.role, is_superadmin=p.is_superadmin
    )
