"""Logika fitur health: kumpulkan hasil pemeriksaan jadi satu laporan."""
import logging

from app.health.domain import HealthReport
from app.health.repository import HealthRepository

log = logging.getLogger("ai-brain")


class HealthService:
    def __init__(self, repo: HealthRepository) -> None:
        self._repo = repo

    def check(self) -> HealthReport:
        try:
            self._repo.ping()
        except Exception as err:  # database mati / salah sandi / tidak terjangkau
            log.warning("health: database tidak bisa dihubungi", extra={"extra_fields": {"error": str(err)[:200]}})
            return HealthReport(status="degraded", database=False, schema_ai=False, pgvector=None)
        return HealthReport(
            status="ok",
            database=True,
            schema_ai=self._repo.schema_ai_exists(),
            pgvector=self._repo.pgvector_version(),
        )
