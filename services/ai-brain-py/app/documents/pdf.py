"""Ekstraksi teks PDF per halaman (pypdf) dan penentuan halaman yang perlu OCR.
Halaman tanpa teks (hasil scan) dikembalikan sebagai string kosong; service memutuskan: OCR bila aktif, selain itu
ditolak/diterima sebagian."""
import io
import re
from collections import Counter

import pypdfium2 as pdfium
import pypdfium2.raw as raw
from pypdf import PdfReader
from pypdf.errors import PyPdfError

# Halaman dianggap "gambar dengan sedikit teks" bila teks isinya (di luar kepala/kaki berulang) kurang dari ini
# DAN gambar menutupi sedikitnya sebagian halaman (mis. formulir yang discan di lampiran SOP).
MIN_BODY_CHARS = 80
MIN_IMAGE_COVERAGE = 0.25


class PdfError(Exception):
    """PDF tidak bisa diproses; pesannya aman ditampilkan ke pengguna."""


def read_pages(data: bytes) -> list[str]:
    """Teks tiap halaman (boleh kosong). PdfError bila rusak, berkata sandi, atau tanpa halaman."""
    try:
        reader = PdfReader(io.BytesIO(data))
        if reader.is_encrypted:
            raise PdfError("PDF dikunci dengan kata sandi")
        pages = [(page.extract_text() or "").strip() for page in reader.pages]
    except PdfError:
        raise
    except (PyPdfError, ValueError, KeyError, OSError) as err:
        raise PdfError("file PDF rusak atau tidak bisa dibaca") from err
    if not pages:
        raise PdfError("PDF tidak berisi halaman")
    return pages


# ---------- halaman yang perlu OCR ----------

def _lines(text: str) -> list[str]:
    return [re.sub(r"\s+", " ", line).strip() for line in text.splitlines() if line.strip()]


def _norm(line: str) -> str:
    """Bentuk pembanding: huruf kecil, setiap deret angka jadi satu '#', tanda baca dibuang ('9 / 13' = '10 / 13')."""
    return re.sub(r"[^a-z#]", "", re.sub(r"\d+", "#", line.lower()))


def _boilerplate(pages: list[str]) -> set[str]:
    """Baris yang berulang di sedikitnya separuh halaman (kepala/kaki dokumen)."""
    if len(pages) < 4:
        return set()
    counts = Counter(norm for text in pages for norm in {_norm(line) for line in _lines(text)} if norm)
    limit = max(3, len(pages) // 2)
    return {norm for norm, n in counts.items() if n >= limit}


def _image_coverage(page: "pdfium.PdfPage") -> float:
    width, height = page.get_size()
    area = 0.0
    for obj in page.get_objects(filter=[raw.FPDF_PAGEOBJ_IMAGE]):
        left, bottom, right, top = obj.get_bounds()
        area += max(0.0, min(right, width) - max(left, 0.0)) * max(0.0, min(top, height) - max(bottom, 0.0))
    return area / (width * height) if width and height else 0.0


def pages_needing_ocr(data: bytes, texts: list[str]) -> list[int]:
    """Indeks halaman (mulai 0) yang sebaiknya di-OCR:
      - halaman tanpa teks sama sekali (hasil scan), dan
      - halaman bergambar besar yang teks isinya sedikit (hanya kepala/kaki berulang), mis. lampiran formulir.
    Bila pdfium gagal membaca file, hanya halaman kosong yang dikembalikan."""
    empty = {i for i, t in enumerate(texts) if not t}
    boiler = _boilerplate(texts)
    short = [i for i, t in enumerate(texts) if t and sum(len(line) for line in _lines(t) if _norm(line) not in boiler) < MIN_BODY_CHARS]
    heavy: set[int] = set()
    if short:
        try:
            pdf = pdfium.PdfDocument(data)
            try:
                heavy = {i for i in short if i < len(pdf) and _image_coverage(pdf[i]) >= MIN_IMAGE_COVERAGE}
            finally:
                pdf.close()
        except Exception:  # noqa: BLE001 - pdfium tidak bisa membuka: abaikan deteksi gambar
            heavy = set()
    return sorted(empty | heavy)


def merge_ocr(existing: str, ocr_text: str) -> str:
    """Teks asli halaman + baris hasil OCR yang belum ada di dalamnya (kepala/kaki yang terbaca dua kali tidak digandakan)."""
    seen = {_norm(line) for line in _lines(existing)}
    extra = [line for line in _lines(ocr_text) if _norm(line) not in seen]
    return "\n".join(part for part in [existing.strip(), *extra] if part)
