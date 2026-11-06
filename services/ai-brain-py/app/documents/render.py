"""Halaman PDF sebagai gambar PNG (pdfium), lengkap dengan tanda air pembaca.
Dipakai tampilan "Asli": tabel, diagram, dan tata letak SOP tampil seperti di PDF-nya."""
import io

import pypdfium2 as pdfium
from PIL import Image, ImageDraw, ImageFont

SCALE = 1.6  # ≈ 115 dpi: cukup jelas dibaca, ukuran berkas masih wajar


def page_count(pdf_bytes: bytes) -> int:
    pdf = pdfium.PdfDocument(pdf_bytes)
    try:
        return len(pdf)
    finally:
        pdf.close()


def _watermark(img: Image.Image, label: str) -> Image.Image:
    """Tulisan diagonal samar di tengah + baris keterangan di kaki gambar."""
    img = img.convert("RGBA")
    w, h = img.size

    big = ImageFont.load_default(size=max(18, w // 22))
    probe = Image.new("RGBA", (w * 2, h // 4), (0, 0, 0, 0))
    d = ImageDraw.Draw(probe)
    d.text((10, 10), label, font=big, fill=(90, 90, 90, 55))
    bbox = probe.getbbox()
    if bbox:
        text_layer = probe.crop(bbox).rotate(30, expand=True, resample=Image.Resampling.BICUBIC)
        img.alpha_composite(text_layer, ((w - text_layer.width) // 2, (h - text_layer.height) // 2))

    small = ImageFont.load_default(size=max(11, w // 70))
    ImageDraw.Draw(img).text((12, h - small.size - 10), f"Dibaca oleh {label}", font=small, fill=(60, 60, 60, 255))
    return img.convert("RGB")


def page_png(pdf_bytes: bytes, page_number: int, label: str) -> bytes:
    """PNG halaman `page_number` (mulai 1) dengan tanda air `label`. IndexError bila halaman tidak ada."""
    pdf = pdfium.PdfDocument(pdf_bytes)
    try:
        if not 1 <= page_number <= len(pdf):
            raise IndexError(page_number)
        img = pdf[page_number - 1].render(scale=SCALE).to_pil()
    finally:
        pdf.close()
    out = io.BytesIO()
    _watermark(img, label).save(out, format="PNG", optimize=False)
    return out.getvalue()
