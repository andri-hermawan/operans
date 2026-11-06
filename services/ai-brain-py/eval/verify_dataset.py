"""Cocokkan set soal evaluasi dengan teks dokumen yang tersimpan di database, supaya tidak ada jawaban karangan.

Dicek per soal:
  - kata_kunci   : SEMUA harus ada di teks halaman yang disebut (spasi/baris baru diabaikan; huruf besar/kecil diabaikan)
  - halaman      : nomor halaman harus ada
  - pemicu_absen : TIDAK BOLEH ada di seluruh teks dokumen (kata utuh), agar soal "tidak ada" benar-benar tidak ada
  - butuh_ocr    : jawabannya hanya ada di gambar. Tanpa --ocr soal ini DILEWATI; dengan --ocr dicek terhadap hasil OCR
                   file asli (halaman yang sama dengan yang akan dibaca OCR saat dokumen diproses)

Jalankan dari folder layanan:
    .venv\\Scripts\\python -m eval.verify_dataset [--ocr] [eval\\sop-hauling-ob.json]
Butuh .env (DATABASE_URL akun ai_app, hanya membaca).
"""
import json
import re
import sys
from collections import Counter
from pathlib import Path

import psycopg

from app.shared.config import get_settings

HERE = Path(__file__).parent


def spaced(text: str) -> str:
    """Huruf kecil, semua spasi/baris baru jadi satu spasi."""
    return re.sub(r"\s+", " ", text.lower()).strip()


def compact(text: str) -> str:
    """Tanpa spasi sama sekali: kata/nomor yang terpotong antar baris tetap cocok ('008-\\n003' = '008-003')."""
    return re.sub(r"\s+", "", text.lower())


def load_pages(dataset: dict) -> tuple[dict[int, str], Path]:
    doc = dataset["dokumen"]
    with psycopg.connect(get_settings().database_url, connect_timeout=5) as conn:
        row = conn.execute("SELECT id, storage_path FROM ai.documents WHERE title = %s AND project_id = %s AND is_active",
                           (doc["judul"], doc["project_id"])).fetchone()
        if row is None:
            sys.exit(f"Dokumen '{doc['judul']}' tidak ditemukan di project {doc['project']}.")
        rows = conn.execute("SELECT page_number, content FROM ai.document_pages WHERE document_id = %s ORDER BY page_number", (row[0],))
        return {n: content for n, content in rows}, Path(get_settings().storage_dir) / row[1]


def ocr_text(pdf_path: Path, texts: dict[int, str], pages: set[int]) -> dict[int, str]:
    """Teks halaman setelah OCR (digabung dengan teks asli), persis seperti yang akan disimpan saat dokumen diproses."""
    from app.documents import pdf
    from app.documents.ocr import RapidOcrEngine

    data = pdf_path.read_bytes()
    result = RapidOcrEngine().read(data, [n - 1 for n in sorted(pages)])
    return {n: pdf.merge_ocr(texts.get(n, ""), result.get(n - 1, "")) for n in pages}


def main() -> int:
    args = [a for a in sys.argv[1:] if not a.startswith("--")]
    with_ocr = "--ocr" in sys.argv
    path = Path(args[0]) if args else HERE / "sop-hauling-ob.json"
    dataset = json.loads(path.read_text(encoding="utf-8"))
    pages, pdf_path = load_pages(dataset)
    ocr_pages: dict[int, str] = {}
    if with_ocr:
        needed = {n for q in dataset["soal"] if q.get("butuh_ocr") for n in q["halaman"]}
        if needed:
            print(f"OCR halaman {sorted(needed)} dari {pdf_path.name} ...")
            ocr_pages = ocr_text(pdf_path, pages, needed)
    skipped: list[str] = []
    whole = spaced(" ".join(pages.values()))
    problems: list[str] = []

    for q in dataset["soal"]:
        if q.get("butuh_ocr") and not with_ocr:
            skipped.append(q["id"])
            continue
        source = ocr_pages if q.get("butuh_ocr") else pages
        where = compact(" ".join(source.get(n, "") for n in q["halaman"]))
        for n in q["halaman"]:
            if n not in pages:
                problems.append(f"{q['id']}: halaman {n} tidak ada (dokumen {len(pages)} halaman)")
        for kw in q["kata_kunci"]:
            if compact(kw) not in where:
                problems.append(f"{q['id']}: kata kunci '{kw}' TIDAK ada di halaman {q['halaman']}")
        for trigger in q.get("pemicu_absen", []):
            if re.search(rf"(?<!\w){re.escape(trigger.strip().lower())}(?!\w)", whole):
                problems.append(f"{q['id']}: kata pemicu '{trigger}' ternyata ADA di dokumen (soal 'tidak ada' keliru)")
        if q["tidak_ada"] is False and not q["kata_kunci"]:
            problems.append(f"{q['id']}: soal yang bisa dijawab harus punya kata_kunci")

    ids = [q["id"] for q in dataset["soal"]]
    if len(ids) != len(set(ids)):
        problems.append("ID soal ganda: " + ", ".join(i for i, n in Counter(ids).items() if n > 1))

    per_kategori = Counter(q["kategori"] for q in dataset["soal"])
    print(f"Dokumen: {dataset['dokumen']['judul']} ({len(pages)} halaman) | soal: {len(ids)}")
    print("Per kategori:", ", ".join(f"{k} {v}" for k, v in per_kategori.items()))
    if skipped:
        print(f"Dilewati (butuh OCR; jalankan dengan --ocr): {', '.join(skipped)}")
    if problems:
        print(f"\n{len(problems)} MASALAH:")
        for p in problems:
            print(" -", p)
        return 1
    print("\nSemua soal cocok dengan teks dokumen.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
