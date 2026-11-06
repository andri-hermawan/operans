"""Pembuat PDF kecil untuk tes (satu string per halaman, boleh berisi beberapa baris dengan \\n;
string kosong = halaman tanpa teks, seperti hasil scan)."""
import io

from PIL import Image
from pypdf import PdfReader, PdfWriter


def make_pdf(pages: list[str]) -> bytes:
    n = len(pages)
    kids = " ".join(f"{4 + 2 * i} 0 R" for i in range(n))
    objs = [
        "<< /Type /Catalog /Pages 2 0 R >>",
        f"<< /Type /Pages /Kids [{kids}] /Count {n} >>",
        "<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>",
    ]
    for i, text in enumerate(pages):
        content = "\n".join(f"BT /F1 12 Tf 72 {720 - 18 * k} Td ({line}) Tj ET" for k, line in enumerate(text.split("\n"))) if text else ""
        objs.append(
            f"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Contents {5 + 2 * i} 0 R"
            " /Resources << /Font << /F1 3 0 R >> >> >>"
        )
        objs.append(f"<< /Length {len(content)} >>\nstream\n{content}\nendstream")

    out = b"%PDF-1.4\n"
    offsets = []
    for i, obj in enumerate(objs, 1):
        offsets.append(len(out))
        out += f"{i} 0 obj\n{obj}\nendobj\n".encode()
    xref_at = len(out)
    out += f"xref\n0 {len(objs) + 1}\n0000000000 65535 f \n".encode()
    for off in offsets:
        out += f"{off:010d} 00000 n \n".encode()
    out += f"trailer\n<< /Size {len(objs) + 1} /Root 1 0 R >>\nstartxref\n{xref_at}\n%%EOF\n".encode()
    return out


def with_big_image(pdf_bytes: bytes, page_index: int) -> bytes:
    """PDF yang sama, dengan gambar besar (±88% halaman) ditumpuk pada satu halaman: meniru lampiran formulir hasil scan."""
    buf = io.BytesIO()
    Image.new("RGB", (600, 700), "lightgray").save(buf, format="PDF", resolution=72)
    overlay = PdfReader(io.BytesIO(buf.getvalue())).pages[0]
    writer = PdfWriter()
    for i, page in enumerate(PdfReader(io.BytesIO(pdf_bytes)).pages):
        added = writer.add_page(page)
        if i == page_index:
            added.merge_page(overlay)
    out = io.BytesIO()
    writer.write(out)
    return out.getvalue()
