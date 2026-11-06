"""Bentuk data fitur documents (hanya struktur, tanpa logika)."""
from datetime import datetime
from typing import Literal
from uuid import UUID

from pydantic import BaseModel, Field

# Admin project & superadmin selalu boleh membaca; role di bawah ini diatur per dokumen.
ReaderRole = Literal["dispatcher", "viewer"]


class Document(BaseModel):
    id: UUID
    department_id: UUID
    department_code: str
    department_name: str
    title: str
    document_number: str | None
    revision: str | None
    file_name: str
    file_size_bytes: int
    page_count: int | None
    processing_status: str  # uploaded | processing | ready | failed
    processing_error: str | None
    created_at: datetime
    roles: list[str] | None = None  # role pembaca; hanya diisi untuk admin


class NewDocument(BaseModel):
    department_id: UUID
    title: str = Field(min_length=1, max_length=200)
    document_number: str | None = Field(default=None, max_length=100)
    revision: str | None = Field(default=None, max_length=30)
    roles: list[ReaderRole] = []


class RolesInput(BaseModel):
    roles: list[ReaderRole]


class PageContent(BaseModel):
    document_id: UUID
    page_number: int
    page_count: int
    content: str
