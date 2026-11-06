"""Tes pembacaan teks PDF per halaman dan penentuan halaman yang perlu OCR."""
import pytest

from app.documents.pdf import PdfError, merge_ocr, pages_needing_ocr, read_pages
from tests.pdfs import make_pdf, with_big_image


def test_teks_per_halaman_terbaca():
    assert read_pages(make_pdf(["Halaman satu", "Halaman dua"])) == ["Halaman satu", "Halaman dua"]


def test_halaman_tanpa_teks_dikembalikan_kosong_bukan_galat():
    assert read_pages(make_pdf(["Ada teks", ""])) == ["Ada teks", ""]
    assert read_pages(make_pdf(["", ""])) == ["", ""]  # keputusan (OCR / tolak) ada di service


def test_file_rusak_ditolak():
    with pytest.raises(PdfError):
        read_pages(b"%PDF-1.4\nbukan pdf sungguhan")


# ---------- halaman yang perlu OCR ----------

HEADER = "RIMS Hauling OB\nSistem Manajemen Terintegrasi\nHalaman {n} / 6"
# isi tiap halaman HARUS berbeda: baris yang sama di separuh halaman dianggap kepala/kaki berulang
KATA = ["alfa", "bravo", "charlie", "delta", "echo", "foxtrot"]
ISIS = [f"Prosedur {k} yang cukup panjang untuk dianggap halaman bertulisan biasa dan bukan lampiran gambar sama sekali." for k in KATA]


def dokumen_enam_halaman(isi_per_halaman: list[str]) -> bytes:
    return make_pdf([HEADER.format(n=i + 1) + ("\n" + isi if isi else "") for i, isi in enumerate(isi_per_halaman)])


def texts_of(pdf: bytes) -> list[str]:
    return read_pages(pdf)


def test_halaman_kosong_selalu_perlu_ocr():
    pdf = make_pdf(["Ada teks", "", "Ada lagi"])
    assert pages_needing_ocr(pdf, texts_of(pdf)) == [1]


def test_dokumen_bertulisan_biasa_tidak_perlu_ocr():
    pdf = dokumen_enam_halaman(ISIS)
    assert pages_needing_ocr(pdf, texts_of(pdf)) == []


def test_halaman_bergambar_besar_dengan_hanya_kepala_kaki_perlu_ocr():
    # halaman 5 (indeks 4): hanya kepala/kaki berulang + gambar besar = lampiran formulir hasil scan
    pdf = with_big_image(dokumen_enam_halaman([*ISIS[:4], "", ISIS[5]]), 4)
    assert pages_needing_ocr(pdf, texts_of(pdf)) == [4]


def test_halaman_pendek_tanpa_gambar_tidak_di_ocr():
    # halaman yang memang hampir kosong ("sengaja dikosongkan") tidak perlu OCR
    pdf = dokumen_enam_halaman([ISIS[0], ISIS[1], "Sengaja dikosongkan", *ISIS[3:]])
    assert pages_needing_ocr(pdf, texts_of(pdf)) == []


def test_halaman_bergambar_tetapi_isi_teks_panjang_tidak_di_ocr():
    pdf = with_big_image(dokumen_enam_halaman(ISIS), 2)  # ada gambar, tetapi teks isinya sudah banyak
    assert pages_needing_ocr(pdf, texts_of(pdf)) == []


def test_nomor_halaman_pada_kaki_tidak_dihitung_sebagai_teks_isi():
    # kaki "Halaman n / 6" berbeda di tiap halaman, tetapi tetap dikenali sebagai kepala/kaki berulang
    pdf = with_big_image(dokumen_enam_halaman([ISIS[0], ISIS[1], "", *ISIS[3:]]), 2)
    assert pages_needing_ocr(pdf, texts_of(pdf)) == [2]


def test_nomor_halaman_satu_dan_dua_angka_dianggap_kaki_yang_sama():
    # "9 / 12" dan "10 / 12": jumlah angkanya berbeda, tetapi tetap kaki berulang (kasus SOP asli: hal. 2-9 vs 10-13)
    kaki = "Dokumen tidak dikendalikan jika tercetak tanpa cap salinan terkendali pada setiap halaman {n} / 12"
    isi = [f"Prosedur {k} yang cukup panjang untuk dianggap halaman bertulisan biasa dan bukan lampiran gambar sama sekali." for k in "abcdefghijkl"]
    isi[9] = ""  # halaman 10: hanya kaki + gambar
    pdf = with_big_image(make_pdf([f"{kaki.format(n=i + 1)}" + (f"\n{t}" if t else "") for i, t in enumerate(isi)]), 9)
    assert pages_needing_ocr(pdf, read_pages(pdf)) == [9]


def test_dokumen_pendek_di_bawah_empat_halaman_tanpa_deteksi_kepala_kaki():
    pdf = make_pdf(["Halaman satu\nTeks", "Halaman dua\nTeks"])
    assert pages_needing_ocr(pdf, texts_of(pdf)) == []  # tanpa gambar: tidak ada yang perlu OCR


# ---------- menggabungkan hasil OCR ----------

def test_merge_ocr_mempertahankan_teks_asli_dan_tidak_menggandakan_kepala():
    existing = "RIMS Hauling OB\nSistem Manajemen Terintegrasi\n7.2. Lampiran"
    ocr = "RIMS Hauling OB\nSistem Manajemen Terintegrasi\nOli Mesin\nRem / Brake\nKlakson"
    assert merge_ocr(existing, ocr) == existing + "\nOli Mesin\nRem / Brake\nKlakson"


def test_merge_ocr_halaman_kosong_hanya_isi_ocr():
    assert merge_ocr("", "baris satu\nbaris dua") == "baris satu\nbaris dua"


def test_merge_ocr_tanda_baca_dan_nomor_halaman_berbeda_tidak_menggandakan():
    assert merge_ocr('Halaman 8 / 13 "SALINAN"', "halaman 9 / 13 SALINAN\nIsi baru") == 'Halaman 8 / 13 "SALINAN"\nIsi baru'
