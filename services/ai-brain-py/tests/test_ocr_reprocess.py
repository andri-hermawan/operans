"""Tes alur OCR (latar belakang) dan proses ulang dokumen. OCR dipalsukan; mesin OCR asli diuji nyata terpisah."""
from uuid import uuid4

import pytest

from app.documents.domain import NewDocument
from app.documents.service import DocumentService
from app.shared.errors import Forbidden, Invalid, NotFound
from tests.pdfs import make_pdf
from tests.test_documents_service import DEPT, PROJECT, Env

SCAN = make_pdf(["", "", ""])
MIXED = make_pdf(["Halaman bertulisan", "", "Halaman bertulisan juga"])


class FakeOcr:
    available = True

    def __init__(self, texts=None, error=None) -> None:
        self.texts, self.error, self.calls = texts or {}, error, []

    def read(self, pdf_bytes, page_indexes):
        self.calls.append(list(page_indexes))
        if self.error:
            raise self.error
        return {i: self.texts.get(i, "") for i in page_indexes}


class OcrEnv(Env):
    def __init__(self, ocr=None) -> None:
        super().__init__()
        self.svc = DocumentService(self.repo, self.audit, self.store, 1024 * 1024, ocr)
        self.tasks: list = []  # tugas latar belakang yang ditunda

    def schedule(self, fn, *args) -> None:
        self.tasks.append((fn, args))

    def run_tasks(self) -> None:
        while self.tasks:
            fn, args = self.tasks.pop(0)
            fn(*args)

    def upload_scan(self, content=SCAN, roles=("viewer",)):
        data = NewDocument(department_id=DEPT, title="SOP Scan", roles=list(roles))
        return self.svc.upload(self.admin, data, "scan.pdf", content, "req", self.schedule)


# ---------- OCR aktif ----------

def test_scan_dengan_ocr_status_processing_lalu_ready_setelah_latar_belakang():
    ocr = FakeOcr({0: "Teks hasil OCR satu", 1: "Teks dua", 2: "Teks tiga"})
    env = OcrEnv(ocr)
    doc = env.upload_scan()
    assert doc.processing_status == "processing" and len(env.tasks) == 1  # belum selesai: OCR ditunda
    env.run_tasks()
    after = env.svc.get(env.admin, doc.id)
    assert after.processing_status == "ready" and after.page_count == 3
    assert env.repo.pages[doc.id] == ["Teks hasil OCR satu", "Teks dua", "Teks tiga"]
    assert ocr.calls == [[0, 1, 2]]
    assert "document_ocr_done" in env.audit.actions()


def test_hanya_halaman_kosong_yang_di_ocr_halaman_bertulisan_tetap_asli():
    ocr = FakeOcr({1: "Isi dari OCR"})
    env = OcrEnv(ocr)
    doc = env.upload_scan(MIXED)
    env.run_tasks()
    assert ocr.calls == [[1]]
    assert env.repo.pages[doc.id] == ["Halaman bertulisan", "Isi dari OCR", "Halaman bertulisan juga"]


def test_dokumen_bertulisan_penuh_tidak_memanggil_ocr():
    ocr = FakeOcr()
    env = OcrEnv(ocr)
    doc = env.upload_scan(make_pdf(["a", "b"]))
    assert doc.processing_status == "ready" and env.tasks == [] and ocr.calls == []


def test_ocr_tidak_menemukan_teks_gagal():
    env = OcrEnv(FakeOcr({}))
    doc = env.upload_scan()
    env.run_tasks()
    after = env.svc.get(env.admin, doc.id)
    assert after.processing_status == "failed" and "OCR tidak menemukan teks" in after.processing_error
    assert len(env.store.files) == 1  # file dipertahankan


def test_ocr_error_tidak_melempar_dari_latar_belakang_dan_ditandai_gagal():
    env = OcrEnv(FakeOcr(error=RuntimeError("model rusak")))
    doc = env.upload_scan()
    env.run_tasks()  # tidak boleh melempar
    after = env.svc.get(env.admin, doc.id)
    assert after.processing_status == "failed" and "OCR gagal" in after.processing_error


# ---------- tanpa OCR ----------

def test_sebagian_halaman_kosong_tanpa_ocr_tetap_diterima():
    env = OcrEnv(None)
    doc = env.upload_scan(MIXED)
    assert doc.processing_status == "ready" and env.repo.pages[doc.id][1] == ""


# ---------- proses ulang ----------

def test_proses_ulang_setelah_ocr_diaktifkan_memulihkan_dokumen_gagal():
    env = OcrEnv(None)
    with pytest.raises(Invalid, match="OCR"):
        env.upload_scan()
    (doc_id,) = env.repo.docs.keys()
    assert env.repo.docs[doc_id]["status"] == "failed"

    env.svc = DocumentService(env.repo, env.audit, env.store, 1024 * 1024, FakeOcr({0: "a", 1: "b", 2: "c"}))  # OCR kini aktif
    doc = env.svc.reprocess(env.admin, doc_id, "r", env.schedule)
    assert doc.processing_status == "processing"
    env.run_tasks()
    assert env.svc.get(env.admin, doc_id).processing_status == "ready"
    assert env.repo.pages[doc_id] == ["a", "b", "c"]
    assert env.audit.actions().count("document_reprocess") == 1


def test_proses_ulang_boleh_untuk_dokumen_ready_dan_tertahan():
    env = OcrEnv(None)
    ready = env.upload(content=make_pdf(["isi"]))
    assert env.svc.reprocess(env.admin, ready.id, "r").processing_status == "ready"  # dokumen siap pun boleh (mis. OCR baru aktif)
    env.repo.docs[ready.id]["status"] = "processing"  # mis. server mati saat OCR berjalan
    assert env.svc.reprocess(env.admin, ready.id, "r").processing_status == "ready"


def test_proses_ulang_dokumen_ready_menambah_teks_dari_ocr_halaman_bergambar():
    from tests.pdfs import with_big_image
    body = [f"Prosedur {k} yang cukup panjang untuk dianggap halaman bertulisan biasa dan bukan lampiran gambar sama sekali." for k in "abcdef"]
    pdf = with_big_image(make_pdf([f"Kepala dokumen\nHalaman {i + 1} / 6" + ("\n" + t if i != 4 else "") for i, t in enumerate(body)]), 4)
    env = OcrEnv(None)  # unggah tanpa OCR: halaman 5 hanya berisi kepala/kaki + gambar
    doc = env.upload_scan(pdf)
    assert doc.processing_status == "ready" and "Oli Mesin" not in env.repo.pages[doc.id][4]

    env.svc = DocumentService(env.repo, env.audit, env.store, 1024 * 1024, FakeOcr({4: "Kepala dokumen\nOli Mesin\nRem / Brake"}))  # OCR kini aktif
    env.svc.reprocess(env.admin, doc.id, "r", env.schedule)
    env.run_tasks()
    page5 = env.repo.pages[doc.id][4]
    assert "Oli Mesin" in page5 and "Rem / Brake" in page5
    assert page5.count("Kepala dokumen") == 1  # kepala yang terbaca dua kali tidak digandakan
    assert env.repo.pages[doc.id][0].startswith("Kepala dokumen")  # halaman lain tidak berubah


def test_proses_ulang_hanya_admin_dan_hanya_project_sendiri():
    env = OcrEnv(None)
    with pytest.raises(Invalid):
        env.upload_scan()
    (doc_id,) = env.repo.docs.keys()
    for p in (env.viewer, env.dispatcher):
        with pytest.raises(Forbidden):
            env.svc.reprocess(p, doc_id, "r")
    from tests.fakes import principal
    with pytest.raises(NotFound):
        env.svc.reprocess(principal(uuid4(), "admin"), doc_id, "r")


def test_proses_ulang_file_hilang_404():
    env = OcrEnv(None)
    with pytest.raises(Invalid):
        env.upload_scan()
    (doc_id,) = env.repo.docs.keys()
    env.store.files.clear()
    with pytest.raises(NotFound, match="penyimpanan"):
        env.svc.reprocess(env.admin, doc_id, "r")


def test_proses_ulang_pdf_masih_tanpa_teks_tetap_gagal_dengan_pesan():
    env = OcrEnv(None)
    with pytest.raises(Invalid):
        env.upload_scan()
    (doc_id,) = env.repo.docs.keys()
    with pytest.raises(Invalid, match="OCR"):
        env.svc.reprocess(env.admin, doc_id, "r")
    assert env.repo.docs[doc_id]["status"] == "failed"
    assert PROJECT  # (project dipakai Env)
