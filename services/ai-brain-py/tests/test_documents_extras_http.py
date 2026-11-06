"""Tes HTTP: gambar halaman, proses ulang, dan unggah scan dengan OCR (latar belakang)."""
from uuid import UUID, uuid4

import pytest

from app.documents import router as doc_router
from app.documents.service import DocumentService
from tests.conftest import PROJECT_ID
from tests.fakes import FakeAudit, FakeDocRepo, FakeStore
from tests.pdfs import make_pdf
from tests.test_ocr_reprocess import SCAN, FakeOcr

PDF = make_pdf(["Isi halaman satu", "Isi halaman dua"])
DEPT = uuid4()


def auth(token: str) -> dict:
    return {"Authorization": f"Bearer {token}"}


def wire(client, ocr=None):
    repo = FakeDocRepo()
    repo.departments.add((UUID(PROJECT_ID), DEPT))
    audit, store = FakeAudit(), FakeStore()
    client.app.dependency_overrides[doc_router.get_service] = lambda: DocumentService(repo, audit, store, 1024 * 1024, ocr)
    client.audit = audit
    return client


@pytest.fixture
def http(client):
    return wire(client)


@pytest.fixture
def http_ocr(client):
    return wire(client, FakeOcr({0: "Hasil OCR halaman satu", 1: "Hasil OCR halaman dua", 2: "Hasil OCR halaman tiga"}))


def upload(http, token, pdf=PDF, roles=("viewer",)):
    return http.post("/api/v1/documents", headers=auth(token),
                     data={"department_id": str(DEPT), "title": "SOP", "roles": list(roles)},
                     files={"file": ("sop.pdf", pdf, "application/pdf")})


# ---------- gambar halaman ----------

def test_viewer_terdaftar_mendapat_gambar_halaman_bertanda_air(http, make_token):
    doc = upload(http, make_token(prole="admin")).json()["id"]
    r = http.get(f"/api/v1/documents/{doc}/pages/2/image", headers=auth(make_token(prole="viewer")))
    assert r.status_code == 200 and r.content.startswith(b"\x89PNG")
    assert r.headers["content-type"] == "image/png" and "no-store" in r.headers["cache-control"]
    assert http.audit.records[-1]["action"] == "document_view" and http.audit.records[-1]["detail"] == {"page": 2, "mode": "image"}


def test_gambar_halaman_dokumen_terlarang_404_halaman_salah_404_tanpa_login_401(http, make_token):
    doc = upload(http, make_token(prole="admin")).json()["id"]
    assert http.get(f"/api/v1/documents/{doc}/pages/1/image", headers=auth(make_token(prole="dispatcher"))).status_code == 404
    for n in (0, 3, 99):
        assert http.get(f"/api/v1/documents/{doc}/pages/{n}/image", headers=auth(make_token(prole="viewer"))).status_code == 404
    assert http.get(f"/api/v1/documents/{doc}/pages/1/image").status_code == 401


def test_file_asli_hilang_gambar_404(http, make_token):
    doc = upload(http, make_token(prole="admin")).json()["id"]
    http.app.dependency_overrides[doc_router.get_service]().__dict__["_store"].files.clear()
    assert http.get(f"/api/v1/documents/{doc}/pages/1/image", headers=auth(make_token(prole="viewer"))).status_code == 404


# ---------- OCR + proses ulang lewat HTTP ----------

def test_unggah_scan_tanpa_ocr_400_lalu_admin_melihat_dokumen_gagal(http, make_token):
    admin = make_token(prole="admin")
    r = upload(http, admin, SCAN)
    assert r.status_code == 400 and "OCR" in r.json()["error"]
    failed = [d for d in http.get("/api/v1/documents", headers=auth(admin)).json() if d["processing_status"] == "failed"]
    assert len(failed) == 1 and "OCR" in failed[0]["processing_error"]


def test_unggah_scan_dengan_ocr_201_processing_lalu_ready(http_ocr, make_token):
    admin = make_token(prole="admin")
    r = upload(http_ocr, admin, SCAN)
    assert r.status_code == 201 and r.json()["processing_status"] == "processing"  # jawaban dikirim sebelum OCR selesai
    got = http_ocr.get(f"/api/v1/documents/{r.json()['id']}", headers=auth(admin)).json()
    assert got["processing_status"] == "ready" and got["page_count"] == 3  # tugas latar belakang sudah selesai
    page = http_ocr.get(f"/api/v1/documents/{r.json()['id']}/pages/1", headers=auth(make_token(prole="viewer")))
    assert page.json()["content"] == "Hasil OCR halaman satu"


def test_proses_ulang_lewat_http_hak_akses_dan_status(http, make_token):
    admin, viewer = make_token(prole="admin"), make_token(prole="viewer")
    ready = upload(http, admin).json()["id"]
    assert http.post(f"/api/v1/documents/{ready}/reprocess", headers=auth(admin)).status_code == 200  # dokumen ready pun boleh
    assert http.post(f"/api/v1/documents/{ready}/reprocess", headers=auth(viewer)).status_code == 403
    assert http.post(f"/api/v1/documents/{ready}/reprocess").status_code == 401
    assert http.post(f"/api/v1/documents/{uuid4()}/reprocess", headers=auth(admin)).status_code == 404


def test_proses_ulang_scan_setelah_ocr_aktif_lewat_http(client, make_token):
    admin = make_token(prole="admin")
    wire(client, None)
    assert upload(client, admin, SCAN).status_code == 400  # gagal: OCR belum aktif
    failed = next(d for d in client.get("/api/v1/documents", headers=auth(admin)).json() if d["processing_status"] == "failed")

    # OCR "dinyalakan": layanan yang sama (repo & penyimpanan sama) kini memakai mesin OCR
    old = client.app.dependency_overrides[doc_router.get_service]()
    ocr = FakeOcr({0: "a", 1: "b", 2: "c"})
    client.app.dependency_overrides[doc_router.get_service] = lambda: DocumentService(old._repo, old._audit, old._store, 1024 * 1024, ocr)
    r = client.post(f"/api/v1/documents/{failed['id']}/reprocess", headers=auth(admin))
    assert r.status_code == 200 and r.json()["processing_status"] == "processing"
    assert client.get(f"/api/v1/documents/{failed['id']}", headers=auth(admin)).json()["processing_status"] == "ready"
