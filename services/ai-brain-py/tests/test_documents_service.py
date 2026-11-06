"""Tes aturan akses & alur unggah dokumen (hak akses sementara: project + role)."""
from uuid import uuid4

import pytest

from app.documents.domain import NewDocument
from app.documents.service import DocumentService
from app.shared.errors import Conflict, Forbidden, Invalid, NotFound, TooLarge
from tests.fakes import FakeAudit, FakeDocRepo, FakeStore, principal
from tests.pdfs import make_pdf

PROJECT = uuid4()
OTHER_PROJECT = uuid4()
DEPT = uuid4()
PDF = make_pdf(["Prosedur kerja aman", "Langkah kedua"])


class Env:
    def __init__(self, max_bytes: int = 1024 * 1024) -> None:
        self.repo, self.audit, self.store = FakeDocRepo(), FakeAudit(), FakeStore()
        self.repo.departments.add((PROJECT, DEPT))
        self.svc = DocumentService(self.repo, self.audit, self.store, max_bytes)
        self.admin = principal(PROJECT, "admin")
        self.dispatcher = principal(PROJECT, "dispatcher")
        self.viewer = principal(PROJECT, "viewer")

    def upload(self, roles=("viewer",), content=PDF, p=None, name="sop.pdf"):
        data = NewDocument(department_id=DEPT, title="SOP Hauling", roles=list(roles))
        return self.svc.upload(p or self.admin, data, name, content, "req-1")


@pytest.fixture
def env() -> Env:
    return Env()


# ---------- unggah ----------

def test_admin_unggah_pdf_jadi_ready_dan_tercatat(env):
    doc = env.upload()
    assert doc.processing_status == "ready" and doc.page_count == 2
    assert env.repo.pages[doc.id][0].startswith("Prosedur")
    assert doc.id and len(env.store.files) == 1
    assert env.audit.actions() == ["document_upload"]


def test_non_admin_tidak_boleh_unggah(env):
    for p in (env.dispatcher, env.viewer):
        with pytest.raises(Forbidden):
            env.upload(p=p)


def test_bukan_pdf_dan_kosong_ditolak(env):
    with pytest.raises(Invalid):
        env.upload(content=b"ini bukan pdf")
    with pytest.raises(Invalid):
        env.upload(content=b"")


def test_terlalu_besar_ditolak():
    small = Env(max_bytes=100)
    with pytest.raises(TooLarge):
        small.upload()


def test_department_project_lain_ditolak(env):
    data = NewDocument(department_id=uuid4(), title="X")
    with pytest.raises(Invalid):
        env.svc.upload(env.admin, data, "x.pdf", PDF, "r")


def test_file_sama_tidak_ganda(env):
    env.upload()
    with pytest.raises(Conflict):
        env.upload()


def test_pdf_tanpa_teks_gagal_file_dipertahankan_dan_tercatat(env):
    scan = make_pdf(["", ""])
    with pytest.raises(Invalid, match="OCR"):
        env.upload(content=scan)
    (d,) = env.repo.docs.values()
    assert d["status"] == "failed" and "OCR" in d["error"]
    assert len(env.store.files) == 1  # file asli dipertahankan agar bisa diproses ulang
    assert env.audit.actions() == ["document_upload_failed"]


def test_pdf_rusak_ditolak(env):
    with pytest.raises(Invalid):
        env.upload(content=b"%PDF-1.4\nrusak total")


def test_superadmin_tanpa_project_harus_memilih_project(env):
    with pytest.raises(Invalid, match="project"):
        env.upload(p=principal(None, None, superadmin=True))


# ---------- akses baca: project + role ----------

def test_viewer_yang_terdaftar_boleh_baca_yang_lain_404(env):
    doc = env.upload(roles=("viewer",))
    assert env.svc.get(env.viewer, doc.id).id == doc.id
    assert [d.id for d in env.svc.list_documents(env.viewer, None)] == [doc.id]
    with pytest.raises(NotFound):  # dispatcher tidak terdaftar → 404, bukan 403
        env.svc.get(env.dispatcher, doc.id)
    assert env.svc.list_documents(env.dispatcher, None) == []


def test_admin_selalu_boleh_walau_tanpa_role_pembaca(env):
    doc = env.upload(roles=())
    assert env.svc.get(env.admin, doc.id).id == doc.id
    with pytest.raises(NotFound):
        env.svc.get(env.viewer, doc.id)


def test_project_lain_tidak_bisa_melihat_termasuk_admin_dan_viewer(env):
    doc = env.upload(roles=("viewer", "dispatcher"))
    for p in (principal(OTHER_PROJECT, "admin"), principal(OTHER_PROJECT, "viewer")):
        with pytest.raises(NotFound):
            env.svc.get(p, doc.id)
        assert env.svc.list_documents(p, None) == []


def test_superadmin_dengan_project_aktif_boleh(env):
    doc = env.upload(roles=())
    sa = principal(PROJECT, None, superadmin=True)
    assert env.svc.get(sa, doc.id).id == doc.id


def test_daftar_role_pembaca_hanya_terlihat_admin(env):
    doc = env.upload(roles=("viewer",))
    assert env.svc.get(env.admin, doc.id).roles == ["viewer"]
    assert env.svc.get(env.viewer, doc.id).roles is None


def test_filter_department(env):
    env.upload()
    assert env.svc.list_documents(env.admin, uuid4()) == []
    assert len(env.svc.list_documents(env.admin, DEPT)) == 1


# ---------- baca halaman ----------

def test_baca_halaman_tercatat_di_audit(env):
    doc = env.upload()
    page = env.svc.read_page(env.viewer, doc.id, 2, "req-9")
    assert page.content == "Langkah kedua" and page.page_count == 2
    assert env.audit.actions()[-1] == "document_view"
    assert env.audit.records[-1]["detail"] == {"page": 2}


def test_halaman_di_luar_rentang_404(env):
    doc = env.upload()
    for n in (0, 3, -1):
        with pytest.raises(NotFound):
            env.svc.read_page(env.viewer, doc.id, n, "r")


def test_dokumen_terlarang_tidak_bisa_dibaca_halamannya(env):
    doc = env.upload(roles=("viewer",))
    with pytest.raises(NotFound):
        env.svc.read_page(env.dispatcher, doc.id, 1, "r")


# ---------- kelola ----------

def test_ubah_role_pembaca_mengubah_akses(env):
    doc = env.upload(roles=("viewer",))
    env.svc.set_roles(env.admin, doc.id, ["dispatcher"], "r")
    assert env.svc.get(env.dispatcher, doc.id).id == doc.id
    with pytest.raises(NotFound):
        env.svc.get(env.viewer, doc.id)
    assert "document_roles_changed" in env.audit.actions()


def test_non_admin_tidak_boleh_ubah_role_atau_tarik(env):
    doc = env.upload()
    with pytest.raises(Forbidden):
        env.svc.set_roles(env.viewer, doc.id, ["viewer"], "r")
    with pytest.raises(Forbidden):
        env.svc.deactivate(env.dispatcher, doc.id, "r")


def test_tarik_dokumen_hilang_dari_semua(env):
    doc = env.upload(roles=("viewer",))
    env.svc.deactivate(env.admin, doc.id, "r")
    with pytest.raises(NotFound):
        env.svc.get(env.admin, doc.id)
    assert env.svc.list_documents(env.viewer, None) == []
