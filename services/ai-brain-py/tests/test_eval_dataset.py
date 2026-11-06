"""Penjaga bentuk set soal evaluasi (tanpa database): ID unik, kategori sah, isian wajib lengkap.
Kecocokan dengan teks dokumen diperiksa terpisah oleh `python -m eval.verify_dataset` (butuh database)."""
import json
from pathlib import Path

import pytest

from eval.verify_dataset import compact, spaced

DATASET = json.loads((Path(__file__).parent.parent / "eval" / "sop-hauling-ob.json").read_text(encoding="utf-8"))
SOAL = DATASET["soal"]
KATEGORI = set(DATASET["keterangan"]["kategori"])


def test_id_unik_dan_berurutan():
    ids = [q["id"] for q in SOAL]
    assert len(ids) == len(set(ids))
    assert ids == [f"Q{i:02d}" for i in range(1, len(ids) + 1)]


@pytest.mark.parametrize("q", SOAL, ids=lambda q: q["id"])
def test_isian_wajib_dan_aturan_per_jenis(q):
    assert q["kategori"] in KATEGORI
    assert q["pertanyaan"].strip().endswith("?") or q["pertanyaan"].strip().endswith(".")
    assert q["jawaban"].strip()
    assert all(1 <= n <= DATASET["dokumen"]["jumlah_halaman"] for n in q["halaman"])
    if q["tidak_ada"]:
        # "tidak ada sama sekali": wajib kata pemicu yang dicek absen. "Sebagian" (rujukan ada, rincian tidak): cukup kata kunci + halaman rujukan.
        assert q.get("pemicu_absen") or (q["kategori"] == "sebagian" and q["kata_kunci"] and q["halaman"])
    else:
        assert q["kata_kunci"] and q["halaman"], "soal yang bisa dijawab harus punya kata kunci dan halaman"


def test_semua_jenis_terwakili_dan_soal_tidak_ada_cukup():
    jumlah = {k: sum(q["kategori"] == k for q in SOAL) for k in KATEGORI}
    assert all(v > 0 for v in jumlah.values()), jumlah
    assert jumlah["tidak-ada"] >= 5  # ukuran terpenting: AI tidak boleh mengarang


def test_pencocok_mengabaikan_spasi_baris_baru_dan_huruf_besar():
    assert compact("RIMS.RMKO-HSE-FRM-008-\n003-R00") == compact("rims.rmko-hse-frm-008-003-r00")
    assert spaced("Beri  prioritas\n pada  ALAT") == "beri prioritas pada alat"
