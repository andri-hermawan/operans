"""Tes fitur search: validasi kata kunci, audit, pemetaan galat HTTP. (Pencocokan & hak akses di SQL diuji nyata ke database.)"""
from uuid import uuid4

import pytest

from app.search import router as search_router
from app.search.domain import SearchHit
from app.search.service import SearchService
from app.shared.errors import Invalid
from tests.conftest import PROJECT_ID
from tests.fakes import FakeAudit, principal

PROJECT = uuid4()


class FakeSearchRepo:
    def __init__(self, hits=None) -> None:
        self.hits = hits or []
        self.calls: list[dict] = []

    def search(self, **kwargs):
        self.calls.append(kwargs)
        return self.hits


def hit() -> SearchHit:
    return SearchHit(document_id=uuid4(), title="SOP Hauling", document_number=None,
                     department_code="OPS", department_name="Operasional", page_number=3,
                     snippet="langkah [[keselamatan]] kerja", score=0.5)


def service(hits=None):
    repo, audit = FakeSearchRepo(hits), FakeAudit()
    return SearchService(repo, audit), repo, audit


def test_pencarian_meneruskan_hak_akses_pemanggil_ke_repository():
    svc, repo, _ = service([hit()])
    svc.search(principal(PROJECT, "viewer"), "  keselamatan  ", None, 10, "r1")
    call = repo.calls[0]
    assert call["project_id"] == PROJECT and call["role"] == "viewer" and call["is_admin"] is False
    assert call["query"] == "keselamatan" and call["limit"] == 10


def test_admin_ditandai_admin():
    svc, repo, _ = service()
    svc.search(principal(PROJECT, "admin"), "apd", None, 5, "r")
    assert repo.calls[0]["is_admin"] is True


def test_hasil_dan_audit():
    svc, _, audit = service([hit(), hit()])
    res = svc.search(principal(PROJECT, "viewer"), "keselamatan", None, 20, "req-5")
    assert res.count == 2 and res.query == "keselamatan"
    assert audit.actions() == ["search"] and audit.records[0]["detail"] == {"q": "keselamatan", "hits": 2}
    assert audit.records[0]["request_id"] == "req-5"


@pytest.mark.parametrize("q", ["", " ", "a", "x" * 201])
def test_kata_kunci_tidak_valid_ditolak_tanpa_menyentuh_database(q):
    svc, repo, audit = service()
    with pytest.raises(Invalid):
        svc.search(principal(PROJECT, "viewer"), q, None, 20, "r")
    assert repo.calls == [] and audit.records == []


@pytest.mark.parametrize("limit", [0, -1, 51])
def test_limit_di_luar_batas_ditolak(limit):
    svc, repo, _ = service()
    with pytest.raises(Invalid):
        svc.search(principal(PROJECT, "viewer"), "apd", None, limit, "r")
    assert repo.calls == []


def test_superadmin_tanpa_project_harus_memilih_project():
    svc, repo, _ = service()
    with pytest.raises(Invalid, match="project"):
        svc.search(principal(None, None, superadmin=True), "apd", None, 20, "r")
    assert repo.calls == []


# ---------- HTTP ----------

@pytest.fixture
def http(client):
    repo = FakeSearchRepo([hit()])
    client.app.dependency_overrides[search_router.get_service] = lambda: SearchService(repo, FakeAudit())
    client.repo = repo
    return client


def auth(token: str) -> dict:
    return {"Authorization": f"Bearer {token}"}


def test_http_butuh_login(http):
    assert http.get("/api/v1/search?q=apd").status_code == 401


def test_http_hasil_pencarian(http, make_token):
    r = http.get("/api/v1/search", params={"q": "keselamatan"}, headers=auth(make_token(prole="viewer")))
    assert r.status_code == 200
    body = r.json()
    assert body["count"] == 1 and body["hits"][0]["page_number"] == 3 and "[[" in body["hits"][0]["snippet"]
    assert str(http.repo.calls[0]["project_id"]) == PROJECT_ID  # project diambil dari TOKEN, bukan dari permintaan


def test_http_tanpa_q_422_dan_terlalu_pendek_400(http, make_token):
    h = auth(make_token(prole="viewer"))
    assert http.get("/api/v1/search", headers=h).status_code == 422
    assert http.get("/api/v1/search?q=a", headers=h).status_code == 400
