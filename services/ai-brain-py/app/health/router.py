"""Pintu HTTP fitur health: GET /health (tanpa login, dipakai pemantau & Nginx)."""
from fastapi import APIRouter, Depends, Response

from app.health.domain import HealthReport
from app.health.repository import HealthRepository
from app.health.service import HealthService
from app.shared.database import Database, get_db

router = APIRouter()


def get_health_service(db: Database = Depends(get_db)) -> HealthService:
    return HealthService(HealthRepository(db))


@router.get("/health", response_model=HealthReport)
def health(response: Response, svc: HealthService = Depends(get_health_service)) -> HealthReport:
    report = svc.check()
    if not report.database:
        response.status_code = 503  # pemantau melihat layanan "tidak sehat"
    return report
