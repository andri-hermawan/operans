"""Tes cetak PDF asli: hanya superadmin; tercatat di audit; file harus ada di penyimpanan."""
import io
from uuid import UUID, uuid4

import pytest

from pypdf import PdfReader

from app.documents import router as doc_router
from app.documents.service import DocumentService
from app.shared.errors import Forbidden, NotFound
from tests.conftest import PROJECT_ID
from tests.fakes import FakeAudit, FakeDocRepo, FakeStore, principal
from tests.test_documents_service import OTHER_PROJECT, PDF, PROJECT, Env


def superadmin(project=PROJECT):
    return principal(project, None, superadmin=True)


# ---------- service ----------

def test_superadmin_mendapat_pdf_asli_dan_tercatat():
    env = Env()
    doc = env.upload(name="SOP Hauling.pdf")
    name, data = env.svc.read_file(superadmin(), doc.id, "req-1")
    assert name == "SOP Hauling.pdf"
    pages = PdfReader(io.BytesIO(data)).pages
    assert len(pages) == 2
    assert "Prosedur kerja aman" in pages[0].extract_text()  # isi asli tetap ada
    assert "DICETAK OLEH UJI" in pages[0].extract_text() and "DICETAK OLEH UJI" in pages[1].extract_text()  # tanda air
    assert env.audit.actions()[-1] == "document_print"
    assert env.audit.records[-1]["detail"] == {"file_name": "SOP Hauling.pdf"}


@pytest.mark.parametrize("who", ["admin", "dispatcher", "viewer"])
def test_selain_superadmin_ditolak_termasuk_admin_project(who):
    env = Env()
    doc = env.upload(roles=("viewer", "dispatcher"))
    with pytest.raises(Forbidden):
        env.svc.read_file(getattr(env, who), doc.id, "r")
    assert "document_print" not in env.audit.actions()


def test_superadmin_project_lain_tidak_bisa_mencetak_dokumen_project_ini():
    env = Env()
    doc = env.upload()
    with pytest.raises(NotFound):
        env.svc.read_file(superadmin(OTHER_PROJECT), doc.id, "r")


def test_dokumen_ditarik_tidak_bisa_dicetak():
    env = Env()
    doc = env.upload()
    env.svc.deactivate(env.admin, doc.id, "r")
    with pytest.raises(NotFound):
        env.svc.read_file(superadmin(), doc.id, "r")


def test_file_hilang_dari_penyimpanan_404_dan_tidak_dicatat():
    env = Env()
    doc = env.upload()
    env.store.files.clear()
    with pytest.raises(NotFound, match="penyimpanan"):
        env.svc.read_file(superadmin(), doc.id, "r")
    assert "document_print" not in env.audit.actions()


def test_superadmin_tanpa_project_harus_memilih_project():
    env = Env()
    with pytest.raises(Exception, match="project"):
        env.svc.read_file(principal(None, None, superadmin=True), uuid4(), "r")


# ---------- HTTP ----------

@pytest.fixture
def http(client):
    repo = FakeDocRepo()
    repo.departments.add((UUID(PROJECT_ID), DEPT))
    audit, store = FakeAudit(), FakeStore()
    client.app.dependency_overrides[doc_router.get_service] = lambda: DocumentService(repo, audit, store, 1024 * 1024)
    client.audit = audit
    return client


DEPT = uuid4()


def auth(token: str) -> dict:
    return {"Authorization": f"Bearer {token}"}


def upload(http, token, pdf=PDF, name="Prosedur Ünik.pdf"):
    r = http.post("/api/v1/documents", headers=auth(token), data={"department_id": str(DEPT), "title": "SOP", "roles": ["viewer"]},
                  files={"file": (name, pdf, "application/pdf")})
    assert r.status_code == 201, r.text
    return r.json()["id"]


def test_http_superadmin_mendapat_pdf_dengan_header_aman(http, make_token):
    doc_id = upload(http, make_token(prole="admin"))
    r = http.get(f"/api/v1/documents/{doc_id}/file", headers=auth(make_token(sa=True, prole=None)))
    assert r.status_code == 200 and r.content.startswith(b"%PDF-")
    assert "DICETAK OLEH UJI" in PdfReader(io.BytesIO(r.content)).pages[0].extract_text()
    assert "Salinan terkendali" in PdfReader(io.BytesIO(r.content)).pages[0].extract_text()
    assert r.headers["content-type"] == "application/pdf"
    assert r.headers["cache-control"] == "no-store" and r.headers["x-content-type-options"] == "nosniff"
    assert r.headers["content-disposition"].startswith("inline; filename*=UTF-8''Prosedur%20")  # nama non-ASCII aman
    assert http.audit.actions()[-1] == "document_print"


def test_http_admin_dan_viewer_403_tanpa_login_401(http, make_token):
    doc_id = upload(http, make_token(prole="admin"))
    for role in ("admin", "dispatcher", "viewer"):
        r = http.get(f"/api/v1/documents/{doc_id}/file", headers=auth(make_token(prole=role)))
        assert r.status_code == 403 and "superadmin" in r.json()["error"]
    assert http.get(f"/api/v1/documents/{doc_id}/file").status_code == 401


def test_http_dokumen_tidak_ada_404(http, make_token):
    r = http.get(f"/api/v1/documents/{uuid4()}/file", headers=auth(make_token(sa=True, prole=None)))
    assert r.status_code == 404

