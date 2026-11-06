"""Tes lapisan HTTP documents & departments: kode status, bentuk galat, unggah multipart."""
from uuid import UUID, uuid4

import pytest

from app.departments import router as dept_router
from app.departments.service import DepartmentService
from app.documents import router as doc_router
from app.documents.service import DocumentService
from tests.conftest import PROJECT_ID
from tests.fakes import FakeAudit, FakeDeptRepo, FakeDocRepo, FakeStore
from tests.pdfs import make_pdf


PDF = make_pdf(["Isi SOP"])
DEPT = uuid4()


def auth(token: str) -> dict:
    return {"Authorization": f"Bearer {token}"}


@pytest.fixture
def http(client):
    repo = FakeDocRepo()
    repo.departments.add((UUID(PROJECT_ID), DEPT))
    store, audit = FakeStore(), FakeAudit()
    client.app.dependency_overrides[doc_router.get_service] = lambda: DocumentService(repo, audit, store, 1024 * 1024)
    depts = FakeDeptRepo()
    client.app.dependency_overrides[dept_router.get_service] = lambda: DepartmentService(depts)
    return client


def form(**over):
    base = {"department_id": str(DEPT), "title": "SOP Hauling", "roles": ["viewer"]}
    base.update(over)
    return base


def upload(http, token, content=PDF, **over):
    return http.post("/api/v1/documents", headers=auth(token), data=form(**over),
                     files={"file": ("sop.pdf", content, "application/pdf")})


def test_semua_endpoint_butuh_login(http):
    for method, path in [("get", "/api/v1/documents"), ("get", "/api/v1/departments"),
                         ("get", f"/api/v1/documents/{uuid4()}"), ("get", f"/api/v1/documents/{uuid4()}/pages/1")]:
        assert getattr(http, method)(path).status_code == 401


def test_admin_unggah_lalu_viewer_membaca(http, make_token):
    admin, viewer = make_token(prole="admin"), make_token(prole="viewer")
    r = upload(http, admin)
    assert r.status_code == 201, r.text
    doc_id = r.json()["id"]
    assert r.json()["roles"] == ["viewer"]

    assert [d["id"] for d in http.get("/api/v1/documents", headers=auth(viewer)).json()] == [doc_id]
    got = http.get(f"/api/v1/documents/{doc_id}", headers=auth(viewer)).json()
    assert got["roles"] is None
    page = http.get(f"/api/v1/documents/{doc_id}/pages/1", headers=auth(viewer))
    assert page.status_code == 200 and page.json()["content"] == "Isi SOP"


def test_viewer_tidak_boleh_unggah_403(http, make_token):
    r = upload(http, make_token(prole="viewer"))
    assert r.status_code == 403 and "admin" in r.json()["error"]


def test_dokumen_tidak_ada_atau_terlarang_404(http, make_token):
    admin, dispatcher = make_token(prole="admin"), make_token(prole="dispatcher")
    doc_id = upload(http, admin).json()["id"]  # hanya role viewer
    assert http.get(f"/api/v1/documents/{doc_id}", headers=auth(dispatcher)).status_code == 404
    assert http.get(f"/api/v1/documents/{uuid4()}", headers=auth(dispatcher)).status_code == 404
    assert http.get(f"/api/v1/documents/{doc_id}/pages/1", headers=auth(dispatcher)).status_code == 404


def test_bukan_pdf_422_atau_400(http, make_token):
    r = upload(http, make_token(prole="admin"), content=b"hanya teks")
    assert r.status_code == 400 and "PDF" in r.json()["error"]


def test_terlalu_besar_413(http, make_token):
    http.app.state.settings.max_upload_mb = 1
    r = upload(http, make_token(prole="admin"), content=b"%PDF-" + b"0" * (1024 * 1024))
    assert r.status_code == 413


def test_file_sama_409(http, make_token):
    admin = make_token(prole="admin")
    assert upload(http, admin).status_code == 201
    assert upload(http, admin).status_code == 409


def test_ubah_role_dan_tarik_dokumen(http, make_token):
    admin, viewer = make_token(prole="admin"), make_token(prole="viewer")
    doc_id = upload(http, admin).json()["id"]
    r = http.put(f"/api/v1/documents/{doc_id}/roles", headers=auth(admin), json={"roles": []})
    assert r.status_code == 200 and r.json()["roles"] == []
    assert http.get(f"/api/v1/documents/{doc_id}", headers=auth(viewer)).status_code == 404
    assert http.delete(f"/api/v1/documents/{doc_id}", headers=auth(admin)).status_code == 204
    assert http.get(f"/api/v1/documents/{doc_id}", headers=auth(admin)).status_code == 404


def test_department_admin_menambah_viewer_hanya_melihat(http, make_token):
    admin, viewer = make_token(prole="admin"), make_token(prole="viewer")
    body = {"department_code": "ops", "department_name": "Operasional Hauling"}
    r = http.post("/api/v1/departments", headers=auth(admin), json=body)
    assert r.status_code == 201 and r.json()["department_code"] == "OPS"
    assert http.post("/api/v1/departments", headers=auth(admin), json=body).status_code == 409
    assert http.post("/api/v1/departments", headers=auth(viewer), json=body).status_code == 403
    assert len(http.get("/api/v1/departments", headers=auth(viewer)).json()) == 1
    bad = {"department_code": "tanpa spasi!", "department_name": "x"}
    assert http.post("/api/v1/departments", headers=auth(admin), json=bad).status_code == 422
