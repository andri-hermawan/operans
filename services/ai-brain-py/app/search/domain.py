"""Bentuk data fitur search."""
from uuid import UUID

from pydantic import BaseModel

# Penanda kata yang cocok di dalam `snippet`. Snippet adalah TEKS BIASA (bukan HTML): tampilan wajib
# meng-escape dulu, baru mengganti penanda ini jadi sorotan.
MARK_START = "[["
MARK_END = "]]"


class SearchHit(BaseModel):
    document_id: UUID
    title: str
    document_number: str | None
    department_code: str
    department_name: str
    page_number: int
    snippet: str  # potongan teks di sekitar kata yang cocok; kata cocok diapit [[ ]]
    score: float  # makin besar makin relevan (hanya untuk mengurutkan)


class SearchResult(BaseModel):
    query: str
    count: int
    hits: list[SearchHit]
