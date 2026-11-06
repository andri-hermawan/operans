"""Tes tanda air cetak (PDF) dan gambar halaman (PNG)."""
import io

import pytest
from PIL import Image
from pypdf import PdfReader

from app.documents import render
from app.documents.watermark import stamp_pdf
from tests.pdfs import make_pdf

PDF = make_pdf(["Isi halaman satu", "Isi halaman dua", "Isi halaman tiga"])


def text_of(pdf: bytes, page: int) -> str:
    return PdfReader(io.BytesIO(pdf)).pages[page].extract_text()


# ---------- tanda air PDF ----------

def test_semua_halaman_diberi_tanda_air_dan_isi_asli_tetap():
    out = stamp_pdf(PDF, "DICETAK OLEH ADI - 04/11/2026 10:00 WIB", "Salinan terkendali OPERANS | SOP | Adi")
    assert len(PdfReader(io.BytesIO(out)).pages) == 3
    for i, asli in enumerate(["Isi halaman satu", "Isi halaman dua", "Isi halaman tiga"]):
        t = text_of(out, i)
        assert asli in t and "DICETAK OLEH ADI" in t and "Salinan terkendali OPERANS" in t


def test_nama_dengan_tanda_kurung_garis_miring_dan_huruf_asing_tidak_merusak_pdf():
    nama = r"BUDI (Ops) \ Ünik 山田"
    out = stamp_pdf(PDF, f"DICETAK OLEH {nama}", f"oleh {nama}")
    assert len(PdfReader(io.BytesIO(out)).pages) == 3
    assert "Ünik" in text_of(out, 0)  # huruf Latin-1 terbaca; huruf di luar WinAnsi diganti "?"


def test_teks_sangat_panjang_tetap_menghasilkan_pdf_valid():
    out = stamp_pdf(PDF, "DICETAK OLEH " + "NAMA PANJANG " * 12, "kaki " * 40)
    assert len(PdfReader(io.BytesIO(out)).pages) == 3


def test_hasil_berbeda_untuk_pencetak_berbeda():
    a = stamp_pdf(PDF, "DICETAK OLEH A", "a")
    b = stamp_pdf(PDF, "DICETAK OLEH B", "b")
    assert "OLEH A" in text_of(a, 0) and "OLEH B" in text_of(b, 0) and a != b


# ---------- gambar halaman ----------

def test_halaman_jadi_png_valid():
    png = render.page_png(PDF, 2, "Adi - 04/11/2026 10:00 WIB")
    assert png.startswith(b"\x89PNG")
    img = Image.open(io.BytesIO(png))
    assert img.mode == "RGB" and img.width > 500 and img.height > img.width  # kertas potret (612x792 pt)


def test_tanda_air_pembaca_mengubah_gambar():
    a = render.page_png(PDF, 1, "Pembaca A")
    b = render.page_png(PDF, 1, "Pembaca B")
    assert a != b


@pytest.mark.parametrize("page", [0, 4, -1])
def test_halaman_di_luar_rentang_indexerror(page):
    with pytest.raises(IndexError):
        render.page_png(PDF, page, "x")


def test_jumlah_halaman():
    assert render.page_count(PDF) == 3
