"""Logika fitur documents: siapa boleh apa, validasi unggahan, alur proses PDF.

Hak akses (sementara: project + role):
  - kelola (unggah, proses ulang, ubah role pembaca, tarik dokumen) : admin project / superadmin
  - cetak PDF asli                                                  : SUPERADMIN saja (bertanda air)
  - baca (daftar, detail, halaman teks/gambar)                      : sesuai filter di repository (project + role pembaca)
  - dokumen yang tidak boleh dilihat -> 404 (bukan 403), agar keberadaannya tidak bocor

Alur proses PDF: baca teks per halaman -> halaman tanpa teks (scan) atau bergambar dengan sedikit teks (lampiran formulir)
di-OCR bila OCR aktif (di latar belakang, status 'processing' sampai selesai) -> 'ready'. File asli selalu disimpan, juga saat gagal, agar bisa diproses ulang.
"""
import hashlib
import logging
from collections.abc import Callable
from uuid import UUID, uuid4

from app.audit.repository import AuditRepository
from app.documents import pdf, render, watermark
from app.documents.domain import Document, NewDocument, PageContent, ReaderRole
from app.documents.ocr import NoOcr, OcrEngine
from app.documents.repository import DocumentRepository
from app.shared.clock import now_wib_text
from app.shared.errors import AppError, Forbidden, Invalid, NotFound, TooLarge
from app.shared.security import Principal
from app.shared.storage import FileStore

log = logging.getLogger("ai-brain")

# Pelaksana tugas latar belakang: (fungsi, *argumen). Router memberi BackgroundTasks.add_task;
# bawaan (tes/pemakaian langsung) menjalankannya saat itu juga.
Schedule = Callable[..., None]


def _run_now(fn: Callable[..., None], *args: object) -> None:
    fn(*args)


class DocumentService:
    def __init__(self, repo: DocumentRepository, audit: AuditRepository, store: FileStore, max_bytes: int,
                 ocr: OcrEngine | None = None) -> None:
        self._repo = repo
        self._audit = audit
        self._store = store
        self._max_bytes = max_bytes
        self._ocr: OcrEngine = ocr or NoOcr()

    # ---------- kelola ----------

    def upload(self, p: Principal, data: NewDocument, file_name: str, content: bytes, request_id: str,
               schedule: Schedule = _run_now) -> Document:
        project_id = p.require_project()
        self._require_admin(p)
        if not content:
            raise Invalid("file kosong")
        if len(content) > self._max_bytes:
            raise TooLarge(f"ukuran file maksimal {self._max_bytes // (1024 * 1024)} MB")
        if not content.startswith(b"%PDF-"):
            raise Invalid("hanya file PDF yang diterima")
        if not self._repo.department_exists(project_id, data.department_id):
            raise Invalid("department tidak ditemukan di project ini")

        doc_id = uuid4()
        storage_path = FileStore.path_for(project_id, doc_id)
        self._repo.create(
            doc_id=doc_id, project_id=project_id, uploader=p.user_id, data=data, file_name=file_name[:255],
            sha256=hashlib.sha256(content).hexdigest(), size=len(content), storage_path=storage_path,
        )
        try:
            self._store.save(storage_path, content)
        except Exception:  # disk penuh / tak bisa ditulis
            self._repo.fail(doc_id, "kesalahan sistem saat menyimpan file")
            raise
        self._process(project_id, p.user_id, doc_id, content, request_id, schedule, "document_upload_failed")
        self._audit.record(project_id=project_id, user_id=p.user_id, action="document_upload", document_id=doc_id,
                           detail={"file_name": file_name[:255], "size": len(content)}, request_id=request_id)
        return self.get(p, doc_id)

    def reprocess(self, p: Principal, doc_id: UUID, request_id: str, schedule: Schedule = _run_now) -> Document:
        """Proses ulang dokumen APA PUN (gagal, tertahan, atau sudah siap; mis. setelah OCR diaktifkan atau diperbaiki)
        memakai file yang sudah tersimpan. Selama diproses, dokumen tidak terlihat oleh pembaca non-admin."""
        project_id = p.require_project()
        self._require_admin(p)
        self.get(p, doc_id)  # 404 bila bukan dokumen aktif di project ini
        info = self._repo.file_info(doc_id)
        content = self._store.read(info[0]) if info else None
        if content is None:
            raise NotFound("file asli tidak ditemukan di penyimpanan")
        self._repo.reset_for_reprocess(doc_id)
        self._audit.record(project_id=project_id, user_id=p.user_id, action="document_reprocess", document_id=doc_id,
                           request_id=request_id)
        self._process(project_id, p.user_id, doc_id, content, request_id, schedule, "document_process_failed")
        return self.get(p, doc_id)

    def set_roles(self, p: Principal, doc_id: UUID, roles: list[ReaderRole], request_id: str) -> Document:
        self._require_admin(p)
        self.get(p, doc_id)  # 404 bila bukan dokumen project ini
        self._repo.set_roles(doc_id, roles)
        self._audit.record(project_id=p.project_id, user_id=p.user_id, action="document_roles_changed",
                           document_id=doc_id, detail={"roles": sorted(set(roles))}, request_id=request_id)
        return self.get(p, doc_id)

    def deactivate(self, p: Principal, doc_id: UUID, request_id: str) -> None:
        self._require_admin(p)
        self.get(p, doc_id)
        self._repo.deactivate(doc_id)
        self._audit.record(project_id=p.project_id, user_id=p.user_id, action="document_deactivated",
                           document_id=doc_id, request_id=request_id)

    # ---------- proses PDF ----------

    def _process(self, project_id: UUID, user_id: UUID, doc_id: UUID, content: bytes, request_id: str,
                 schedule: Schedule, failed_action: str) -> None:
        """Baca teks; halaman tanpa teks di-OCR (latar belakang) bila aktif. Gagal -> status 'failed' + Invalid."""
        try:
            pages = pdf.read_pages(content)
        except pdf.PdfError as err:
            self._mark_failed(project_id, user_id, doc_id, str(err), request_id, failed_action)
            raise Invalid(str(err)) from None

        try:
            if self._ocr.available:
                blank = pdf.pages_needing_ocr(content, pages)
                if blank:
                    schedule(self._finish_with_ocr, project_id, user_id, doc_id, content, pages, blank, request_id)
                    return  # status tetap 'processing' sampai OCR selesai
            if not any(pages):
                message = "tidak ada teks yang terbaca (kemungkinan hasil scan; OCR belum diaktifkan)"
                self._mark_failed(project_id, user_id, doc_id, message, request_id, failed_action)
                raise Invalid(message)
            self._repo.finish(doc_id, pages)  # sebagian halaman tanpa teks tetap diterima
        except AppError:
            raise
        except Exception:  # database putus, dsb.
            self._repo.fail(doc_id, "kesalahan sistem saat memproses file")
            raise

    def _finish_with_ocr(self, project_id: UUID, user_id: UUID, doc_id: UUID, content: bytes, pages: list[str],
                         blank: list[int], request_id: str) -> None:
        """Berjalan di latar belakang: tidak boleh melempar galat (tidak ada yang menangkapnya)."""
        try:
            texts = self._ocr.read(content, blank)
            for i in blank:
                pages[i] = pdf.merge_ocr(pages[i], texts.get(i) or "")  # teks asli halaman dipertahankan
            if not any(pages):
                self._mark_failed(project_id, user_id, doc_id, "OCR tidak menemukan teks pada dokumen ini", request_id,
                                  "document_process_failed")
                return
            self._repo.finish(doc_id, pages)
            self._audit.record(project_id=project_id, user_id=user_id, action="document_ocr_done", document_id=doc_id,
                               detail={"pages_ocr": len(blank), "pages": len(pages)}, request_id=request_id)
        except Exception:
            log.exception("OCR gagal", extra={"extra_fields": {"document_id": str(doc_id)}})
            try:
                self._repo.fail(doc_id, "OCR gagal; coba proses ulang")
            except Exception:
                log.exception("status gagal tidak bisa disimpan")

    def _mark_failed(self, project_id: UUID, user_id: UUID, doc_id: UUID, message: str, request_id: str, action: str) -> None:
        """Tandai gagal; file asli SENGAJA dipertahankan agar bisa diproses ulang."""
        self._repo.fail(doc_id, message)
        self._audit.record(project_id=project_id, user_id=user_id, action=action, document_id=doc_id,
                           detail={"error": message}, request_id=request_id)

    # ---------- cetak ----------

    def read_file(self, p: Principal, doc_id: UUID, request_id: str) -> tuple[str, bytes]:
        """PDF asli untuk dicetak, BERTANDA AIR (pencetak + waktu). HANYA superadmin; dokumen harus di project aktif."""
        if not p.is_superadmin:
            raise Forbidden("hanya superadmin yang boleh mencetak dokumen")
        doc = self.get(p, doc_id)  # 404 bila bukan dokumen aktif di project ini
        info = self._repo.file_info(doc_id)
        data = self._store.read(info[0]) if info else None
        if info is None or data is None:
            raise NotFound("file asli tidak ditemukan di penyimpanan")
        who, when = p.name or "superadmin", now_wib_text()
        try:
            stamped = watermark.stamp_pdf(
                data,
                headline=f"DICETAK OLEH {who.upper()} - {when}",
                footer=f"Salinan terkendali OPERANS | {doc.title} | dicetak oleh {who} | {when}",
            )
        except Exception:  # jangan pernah mengirim salinan tanpa tanda air
            log.exception("tanda air gagal", extra={"extra_fields": {"document_id": str(doc_id)}})
            raise Invalid("PDF ini tidak bisa diberi tanda air, sehingga tidak dapat dicetak") from None
        self._audit.record(project_id=p.project_id, user_id=p.user_id, action="document_print", document_id=doc_id,
                           detail={"file_name": info[1]}, request_id=request_id)
        return info[1], stamped

    # ---------- baca ----------

    def list_documents(self, p: Principal, department_id: UUID | None) -> list[Document]:
        docs = self._repo.find_visible(project_id=p.require_project(), is_admin=p.is_admin, role=p.role,
                                       department_id=department_id)
        return [self._hide_roles(p, d) for d in docs]

    def get(self, p: Principal, doc_id: UUID) -> Document:
        doc = self._repo.get(project_id=p.require_project(), doc_id=doc_id, is_admin=p.is_admin, role=p.role)
        if doc is None:
            raise NotFound("dokumen tidak ditemukan")
        return self._hide_roles(p, doc)

    def read_page(self, p: Principal, doc_id: UUID, page_number: int, request_id: str) -> PageContent:
        doc = self.get(p, doc_id)
        content = self._repo.get_page(doc_id, page_number) if doc.page_count else None
        if content is None:
            raise NotFound("halaman tidak ditemukan")
        self._audit.record(project_id=p.project_id, user_id=p.user_id, action="document_view", document_id=doc_id,
                           detail={"page": page_number}, request_id=request_id)
        return PageContent(document_id=doc_id, page_number=page_number, page_count=doc.page_count or 0, content=content)

    def read_page_image(self, p: Principal, doc_id: UUID, page_number: int, request_id: str) -> bytes:
        """Halaman sebagai gambar PNG bertanda air pembaca. Hak akses sama dengan membaca teks halaman."""
        doc = self.get(p, doc_id)
        if not doc.page_count or not 1 <= page_number <= doc.page_count:
            raise NotFound("halaman tidak ditemukan")
        info = self._repo.file_info(doc_id)
        data = self._store.read(info[0]) if info else None
        if data is None:
            raise NotFound("file asli tidak ditemukan di penyimpanan")
        try:
            png = render.page_png(data, page_number, f"{p.name or 'pengguna'} - {now_wib_text()}")
        except IndexError:
            raise NotFound("halaman tidak ditemukan") from None
        self._audit.record(project_id=p.project_id, user_id=p.user_id, action="document_view", document_id=doc_id,
                           detail={"page": page_number, "mode": "image"}, request_id=request_id)
        return png

    # ---------- bantuan ----------

    @staticmethod
    def _require_admin(p: Principal) -> None:
        if not p.is_admin:
            raise Forbidden("hanya admin yang boleh mengelola dokumen")

    @staticmethod
    def _hide_roles(p: Principal, doc: Document) -> Document:
        """Daftar role pembaca hanya untuk admin."""
        return doc if p.is_admin else doc.model_copy(update={"roles": None})
