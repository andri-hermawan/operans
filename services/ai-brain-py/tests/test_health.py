"""Tes /health: laporan sesuai kondisi database; tidak butuh login."""
from app.health.domain import HealthReport
from app.health.repository import HealthRepository
from app.health.router import get_health_service
from app.health.service import HealthService


class FakeRepo(HealthRepository):
    def __init__(self, *, up: bool, schema: bool = True, vector: str | None = None) -> None:
        self.up, self.schema, self.vector = up, schema, vector

    def ping(self) -> None:
        if not self.up:
            raise ConnectionError("database mati")

    def schema_ai_exists(self) -> bool:
        return self.schema

    def pgvector_version(self) -> str | None:
        return self.vector


def with_repo(client, repo: FakeRepo) -> None:
    client.app.dependency_overrides[get_health_service] = lambda: HealthService(repo)


def test_database_hidup_status_ok(client):
    with_repo(client, FakeRepo(up=True, schema=True, vector="0.8.7"))
    r = client.get("/health")
    assert r.status_code == 200
    assert HealthReport(**r.json()) == HealthReport(status="ok", database=True, schema_ai=True, pgvector="0.8.7")


def test_pgvector_belum_aktif_tetap_ok(client):
    with_repo(client, FakeRepo(up=True, schema=False, vector=None))
    r = client.get("/health")
    assert r.status_code == 200 and r.json()["pgvector"] is None and r.json()["schema_ai"] is False


def test_database_mati_503_degraded(client):
    with_repo(client, FakeRepo(up=False))
    r = client.get("/health")
    assert r.status_code == 503
    assert r.json()["status"] == "degraded" and r.json()["database"] is False


def test_galat_berbentuk_error_seragam(client):
    r = client.get("/tidak-ada")
    assert r.status_code == 404 and "error" in r.json()
