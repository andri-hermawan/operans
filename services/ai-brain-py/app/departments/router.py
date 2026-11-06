"""Pintu HTTP fitur departments: /api/v1/departments."""
from fastapi import APIRouter, Depends

from app.departments.domain import Department, DepartmentInput
from app.departments.repository import DepartmentRepository
from app.departments.service import DepartmentService
from app.shared.database import Database, get_db
from app.shared.security import Principal, current_principal

router = APIRouter(prefix="/api/v1/departments", tags=["departments"])


def get_service(db: Database = Depends(get_db)) -> DepartmentService:
    return DepartmentService(DepartmentRepository(db))


@router.get("", response_model=list[Department])
def list_departments(p: Principal = Depends(current_principal), svc: DepartmentService = Depends(get_service)):
    return svc.list_departments(p)


@router.post("", response_model=Department, status_code=201)
def create_department(
    data: DepartmentInput, p: Principal = Depends(current_principal), svc: DepartmentService = Depends(get_service)
):
    return svc.create(p, data)
