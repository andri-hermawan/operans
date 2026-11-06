"""Tanda air pada PDF yang dicetak: tulisan diagonal samar di tiap halaman + baris keterangan di kaki halaman.
Tujuannya penelusuran: salinan cetak selalu memuat siapa yang mencetak dan kapan.

Tanda air dibuat sebagai PDF satu halaman (tulisan Helvetica bawaan PDF, tanpa file huruf) lalu ditumpuk ke
setiap halaman dengan pypdf, jadi tidak perlu pustaka tambahan.
"""
import io
import math

from pypdf import PdfReader, PdfWriter

_cache: dict[tuple[int, int, str, str], bytes] = {}


def _pdf_text(s: str) -> str:
    """Teks aman untuk string PDF: huruf di luar WinAnsi diganti '?', tanda kurung/garis miring di-escape."""
    safe = s.encode("cp1252", errors="replace").decode("cp1252")
    return safe.replace("\\", "\\\\").replace("(", "\\(").replace(")", "\\)")


def _overlay(width: float, height: float, headline: str, footer: str) -> bytes:
    key = (round(width), round(height), headline, footer)
    if key in _cache:
        return _cache[key]

    diagonal = math.hypot(width, height)
    size = max(14.0, min(48.0, 0.8 * diagonal / (max(len(headline), 1) * 0.55)))  # muat di diagonal halaman
    text_len = 0.55 * size * len(headline)  # lebar kira-kira teks Helvetica
    angle = math.atan2(height, width)
    c, s = math.cos(angle), math.sin(angle)
    x0 = width / 2 - (text_len / 2) * c + (size / 3) * s
    y0 = height / 2 - (text_len / 2) * s - (size / 3) * c

    content = (
        f"q /GS1 gs 0.45 g BT /F1 {size:.1f} Tf {c:.4f} {s:.4f} {-s:.4f} {c:.4f} {x0:.1f} {y0:.1f} Tm "
        f"({_pdf_text(headline)}) Tj ET Q\n"
        f"q 0.25 g BT /F1 7.5 Tf 24 12 Td ({_pdf_text(footer)}) Tj ET Q"
    )
    objs = [
        "<< /Type /Catalog /Pages 2 0 R >>",
        "<< /Type /Pages /Kids [4 0 R] /Count 1 >>",
        "<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica /Encoding /WinAnsiEncoding >>",
        f"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 {width:.2f} {height:.2f}] /Contents 5 0 R"
        " /Resources << /Font << /F1 3 0 R >> /ExtGState << /GS1 << /ca 0.22 >> >> >> >>",
        f"<< /Length {len(content.encode('cp1252'))} >>\nstream\n{content}\nendstream",
    ]
    out = b"%PDF-1.4\n"
    offsets = []
    for i, obj in enumerate(objs, 1):
        offsets.append(len(out))
        out += f"{i} 0 obj\n{obj}\nendobj\n".encode("cp1252")
    xref_at = len(out)
    out += f"xref\n0 {len(objs) + 1}\n0000000000 65535 f \n".encode()
    out += b"".join(f"{off:010d} 00000 n \n".encode() for off in offsets)
    out += f"trailer\n<< /Size {len(objs) + 1} /Root 1 0 R >>\nstartxref\n{xref_at}\n%%EOF\n".encode()
    if len(_cache) > 64:
        _cache.clear()
    _cache[key] = out
    return out


def stamp_pdf(data: bytes, headline: str, footer: str) -> bytes:
    """PDF yang sama dengan `headline` (diagonal) dan `footer` di setiap halaman."""
    reader = PdfReader(io.BytesIO(data))
    writer = PdfWriter()
    for page in reader.pages:
        overlay = PdfReader(io.BytesIO(_overlay(float(page.mediabox.width), float(page.mediabox.height), headline, footer)))
        writer.add_page(page).merge_page(overlay.pages[0])  # tumpuk setelah halaman terpasang ke writer
    out = io.BytesIO()
    writer.write(out)
    return out.getvalue()
