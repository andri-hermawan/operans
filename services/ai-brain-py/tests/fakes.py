"""Pengganti repository/penyimpanan/audit di memori, supaya aturan akses & alur unggah dites tanpa database.
(Filter SQL aslinya diuji lewat uji nyata ke database — lihat README.)"""
from datetime import datetime, timezone
from uuid import UUID, uuid4

from app.departments.domain import Department, DepartmentInput
from app.documents.domain import Document, NewDocument
from app.shared.errors import Conflict
from app.shared.security import Principal


class FakeDocRepo:
    def __init__(self) -> None:
        self.docs: dict[UUID, dict] = {}
        self.pages: dict[UUID, list[str]] = {}
        self.departments: set[tuple[UUID, UUID]] = set()  # (project_id, department_id)

    def department_exists(self, project_id, department_id) -> bool:
        return (project_id, department_id) in self.departments

    def create(self, *, doc_id, project_id, uploader, data: NewDocument, file_name, sha256, size, storage_path):
        if any(d["project_id"] == project_id and d["sha256"] == sha256 for d in self.docs.values()):
            raise Conflict("file yang sama sudah pernah diunggah di project ini")
        self.docs[doc_id] = dict(id=doc_id, project_id=project_id, data=data, file_name=file_name, sha256=sha256,
                                 size=size, storage_path=storage_path, status="processing", error=None,
                                 page_count=None, roles=set(data.roles), active=True)

    def finish(self, doc_id, pages):
        self.pages[doc_id] = pages
        self.docs[doc_id].update(status="ready", page_count=len(pages))

    def reset_for_reprocess(self, doc_id):
        self.pages.pop(doc_id, None)
        self.docs[doc_id].update(status="processing", error=None, page_count=None)

    def fail(self, doc_id, message):
        self.docs[doc_id].update(status="failed", error=message)

    def _visible(self, d, project_id, is_admin, role) -> bool:
        if d["project_id"] != project_id or not d["active"]:
            return False
        return is_admin or (d["status"] == "ready" and role in d["roles"])

    def _to_doc(self, d) -> Document:
        data: NewDocument = d["data"]
        return Document(
            id=d["id"], department_id=data.department_id, department_code="OPS", department_name="Operasional",
            title=data.title, document_number=data.document_number,
            revision=data.revision, file_name=d["file_name"], file_size_bytes=d["size"], page_count=d["page_count"],
            processing_status=d["status"], processing_error=d["error"], created_at=datetime.now(timezone.utc),
            roles=sorted(d["roles"]),
        )

    def find_visible(self, *, project_id, is_admin, role, department_id):
        return [self._to_doc(d) for d in self.docs.values()
                if self._visible(d, project_id, is_admin, role)
                and (department_id is None or d["data"].department_id == department_id)]

    def get(self, *, project_id, doc_id, is_admin, role):
        d = self.docs.get(doc_id)
        return self._to_doc(d) if d and self._visible(d, project_id, is_admin, role) else None

    def file_info(self, doc_id):
        d = self.docs.get(doc_id)
        return (d["storage_path"], d["file_name"]) if d else None

    def get_page(self, doc_id, page_number):
        pages = self.pages.get(doc_id, [])
        return pages[page_number - 1] if 1 <= page_number <= len(pages) else None

    def set_roles(self, doc_id, roles):
        self.docs[doc_id]["roles"] = set(roles)

    def deactivate(self, doc_id):
        self.docs[doc_id]["active"] = False


class FakeAudit:
    def __init__(self) -> None:
        self.records: list[dict] = []

    def record(self, **kwargs) -> None:
        self.records.append(kwargs)

    def actions(self) -> list[str]:
        return [r["action"] for r in self.records]


class FakeStore:
    def __init__(self) -> None:
        self.files: dict[str, bytes] = {}

    def save(self, rel: str, data: bytes) -> None:
        self.files[rel] = data

    def read(self, rel: str):
        return self.files.get(rel)

    def delete(self, rel: str) -> None:
        self.files.pop(rel, None)


class FakeDeptRepo:
    def __init__(self) -> None:
        self.items: list[tuple[UUID, Department]] = []

    def list_active(self, project_id):
        return [d for pid, d in self.items if pid == project_id]

    def create(self, project_id, data: DepartmentInput):
        if any(pid == project_id and d.department_code == data.department_code.upper() for pid, d in self.items):
            raise Conflict("kode department sudah dipakai di project ini")
        dept = Department(id=uuid4(), department_code=data.department_code.upper(), department_name=data.department_name)
        self.items.append((project_id, dept))
        return dept


def principal(project_id: UUID | None, role: str | None = "viewer", *, superadmin: bool = False) -> Principal:
    return Principal(user_id=uuid4(), name="Uji", project_id=project_id, role=role, is_superadmin=superadmin)
