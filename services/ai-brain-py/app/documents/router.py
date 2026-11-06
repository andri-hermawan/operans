"""Pintu HTTP fitur documents: /api/v1/documents (terima request -> panggil service -> kirim JSON)."""
from urllib.parse import quote
from uuid import UUID

from fastapi import APIRouter, BackgroundTasks, Depends, File, Form, Request, Response, UploadFile

from app.audit.repository import AuditRepository
from app.documents.domain import Document, NewDocument, PageContent, ReaderRole, RolesInput
from app.documents.repository import DocumentRepository
from app.documents.service import DocumentService
from app.shared.database import Database, get_db
from app.shared.errors import TooLarge
from app.shared.security import Principal, current_principal

router = APIRouter(prefix="/api/v1/documents", tags=["documents"])


def get_service(request: Request, db: Database = Depends(get_db)) -> DocumentService:
    max_bytes = request.app.state.settings.max_upload_mb * 1024 * 1024
    return DocumentService(DocumentRepository(db), AuditRepository(db), request.app.state.store, max_bytes, request.app.state.ocr)


def _request_id(request: Request) -> str:
    return request.state.request_id


@router.post("", response_model=Document, status_code=201)
def upload_document(
    request: Request,
    background: BackgroundTasks,
    file: UploadFile = File(...),
    department_id: UUID = Form(...),
    title: str = Form(...),
    document_number: str | None = Form(None),
    revision: str | None = Form(None),
    roles: list[ReaderRole] = Form([]),
    p: Principal = Depends(current_principal),
    svc: DocumentService = Depends(get_service),
):
    max_mb = request.app.state.settings.max_upload_mb
    content = file.file.read(max_mb * 1024 * 1024 + 1)  # baca sampai batas + 1: cukup untuk tahu "terlalu besar"
    if len(content) > max_mb * 1024 * 1024:
        raise TooLarge(f"ukuran file maksimal {max_mb} MB")
    data = NewDocument(department_id=department_id, title=title,
                       document_number=document_number or None, revision=revision or None, roles=roles)
    return svc.upload(p, data, file.filename or "dokumen.pdf", content, _request_id(request), background.add_task)


@router.get("", response_model=list[Document])
def list_documents(
    department_id: UUID | None = None,
    p: Principal = Depends(current_principal),
    svc: DocumentService = Depends(get_service),
):
    return svc.list_documents(p, department_id)


@router.get("/{doc_id}", response_model=Document)
def get_document(doc_id: UUID, p: Principal = Depends(current_principal), svc: DocumentService = Depends(get_service)):
    return svc.get(p, doc_id)


@router.get("/{doc_id}/pages/{page_number}", response_model=PageContent)
def read_page(
    request: Request, doc_id: UUID, page_number: int,
    p: Principal = Depends(current_principal), svc: DocumentService = Depends(get_service),
):
    return svc.read_page(p, doc_id, page_number, _request_id(request))


@router.get("/{doc_id}/pages/{page_number}/image")
def read_page_image(
    request: Request, doc_id: UUID, page_number: int,
    p: Principal = Depends(current_principal), svc: DocumentService = Depends(get_service),
):
    """Halaman sebagai gambar PNG bertanda air pembaca (hak akses sama dengan membaca teks halaman)."""
    png = svc.read_page_image(p, doc_id, page_number, _request_id(request))
    return Response(content=png, media_type="image/png",
                    headers={"Cache-Control": "private, no-store", "X-Content-Type-Options": "nosniff"})


@router.post("/{doc_id}/reprocess", response_model=Document)
def reprocess_document(
    request: Request, doc_id: UUID, background: BackgroundTasks,
    p: Principal = Depends(current_principal), svc: DocumentService = Depends(get_service),
):
    """Proses ulang dokumen yang gagal/tertahan (admin) memakai file yang sudah tersimpan."""
    return svc.reprocess(p, doc_id, _request_id(request), background.add_task)


@router.get("/{doc_id}/file")
def print_file(
    request: Request, doc_id: UUID, p: Principal = Depends(current_principal), svc: DocumentService = Depends(get_service)
):
    """PDF asli untuk dicetak, bertanda air — hanya superadmin; tiap pencetakan dicatat di audit."""
    name, data = svc.read_file(p, doc_id, _request_id(request))
    return Response(
        content=data,
        media_type="application/pdf",
        headers={
            "Content-Disposition": f"inline; filename*=UTF-8''{quote(name)}",
            "Cache-Control": "no-store",
            "X-Content-Type-Options": "nosniff",
        },
    )


@router.put("/{doc_id}/roles", response_model=Document)
def set_roles(
    request: Request, doc_id: UUID, body: RolesInput,
    p: Principal = Depends(current_principal), svc: DocumentService = Depends(get_service),
):
    return svc.set_roles(p, doc_id, body.roles, _request_id(request))


@router.delete("/{doc_id}", status_code=204)
def deactivate_document(
    request: Request, doc_id: UUID, p: Principal = Depends(current_principal), svc: DocumentService = Depends(get_service)
):
    svc.deactivate(p, doc_id, _request_id(request))
    return Response(status_code=204)
