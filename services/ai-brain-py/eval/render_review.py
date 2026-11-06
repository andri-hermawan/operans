"""Buat berkas tinjauan docs/ai-evaluasi-sop.md dari set soal JSON (satu sumber: JSON, jangan edit .md dengan tangan).

    .venv\\Scripts\\python -m eval.render_review [eval\\sop-hauling-ob.json]
"""
import json
import sys
from collections import Counter
from pathlib import Path

HERE = Path(__file__).parent
OUT = HERE.parent.parent.parent / "docs" / "ai-evaluasi-sop.md"

TITLES = {
    "fakta": "A. Fakta langsung",
    "parafrase": "B. Pertanyaan dengan kata-kata berbeda (menguji pencarian makna)",
    "multi-halaman": "C. Jawaban dari beberapa bagian",
    "sebagian": "D. Topiknya ada, rinciannya tidak ada",
    "tidak-ada": "E. Tidak ada di dokumen (AI harus menjawab tidak ditemukan)",
    "dalam-gambar": "F. Jawaban hanya ada di halaman gambar (butuh OCR)",
    "ambigu": "G. Dokumen sendiri tidak konsisten",
}


def cell(text: str) -> str:
    return text.replace("|", "\\|").replace("\n", " ")


def main() -> None:
    path = Path(sys.argv[1]) if len(sys.argv) > 1 else HERE / "sop-hauling-ob.json"
    d = json.loads(path.read_text(encoding="utf-8"))
    doc, soal = d["dokumen"], d["soal"]
    counts = Counter(q["kategori"] for q in soal)

    lines = [
        f"# Set Soal Evaluasi AI — SOP {doc['judul']}",
        "",
        f"**Status: DRAF, menunggu tinjauan.** Dokumen: *{doc['nomor']}* ({doc['jumlah_halaman']} halaman, project {doc['project']}).",
        "",
        "Ini \"soal ujian\" untuk AI SOP. Nanti, setiap kali model atau pengaturan diganti, soal yang sama dijalankan dan jawaban AI",
        "dibandingkan dengan **jawaban yang benar** di bawah, sehingga kualitasnya terukur dengan angka, bukan perasaan.",
        "",
        "Semua jawaban sudah dicocokkan otomatis dengan teks dokumen di database (kata kunci jawaban ada di halaman yang disebut;",
        "soal \"tidak ada\" memang tidak memuat kata terkait di seluruh dokumen). Yang belum bisa dicek mesin: **apakah jawaban",
        "dan pertanyaannya masuk akal untuk kerja lapangan.** Itu tugas Anda.",
        "",
        "## Cara meninjau",
        "",
        "1. Baca tiap baris. Bandingkan **jawaban yang benar** dengan SOP aslinya (nomor halaman ada di kolom *Hal.*).",
        "2. Tandai kolom **Tinjauan**: ✔ benar · ✎ perlu diperbaiki (tulis perbaikannya) · ✘ hapus soal.",
        "3. Tambahkan 10–15 soal yang **benar-benar sering ditanyakan di lapangan** (istilah yang dipakai operator, bukan istilah dokumen).",
        "4. Kirim hasilnya; saya perbarui `services/ai-brain-py/eval/sop-hauling-ob.json` (sumber data) lalu berkas ini dibuat ulang.",
        "",
        "## Ringkasan",
        "",
        "| Jenis soal | Jumlah | Fungsinya |",
        "|---|---|---|",
    ]
    for key, title in TITLES.items():
        if counts.get(key):
            lines.append(f"| {title.split('. ', 1)[1]} | {counts[key]} | {d['keterangan']['kategori'][key]} |")
    lines += [f"| **Total** | **{len(soal)}** | |", "", "## Temuan dari dokumennya", ""]
    lines += [f"- {n}" for n in d["catatan_sumber"]]
    lines += ["", "> Soal kategori E dan sebagian D paling penting: AI yang baik **menolak mengarang** bila SOP tidak memuat jawabannya.", ""]

    for key, title in TITLES.items():
        rows = [q for q in soal if q["kategori"] == key]
        if not rows:
            continue
        lines += [f"## {title}", "", "| ID | Pertanyaan | Jawaban yang benar | Hal. | Tinjauan | Catatan |", "|---|---|---|---|---|---|"]
        for q in rows:
            halaman = ", ".join(str(n) for n in q["halaman"]) or "—"
            lines.append(f"| {q['id']} | {cell(q['pertanyaan'])} | {cell(q['jawaban'])} | {halaman} | ☐ | {cell(q.get('catatan', ''))} |")
        lines.append("")

    lines += [
        "## Rencana pemakaian (setelah soal ini ditinjau)",
        "",
        "| Ukuran | Target awal |",
        "|---|---|",
        "| Soal A–C dijawab benar dengan rujukan halaman yang tepat | ≥ 90% |",
        "| Soal D–E (tidak ada) dijawab \"tidak ditemukan\", tanpa karangan | **100%** |",
        "| Soal F disebut kedua versi / ketidakkonsistenan | dicatat saja |",
        "",
        "Berkas ini dibuat otomatis oleh `python -m eval.render_review`; edit sumbernya di `eval/sop-hauling-ob.json`.",
        "",
    ]
    OUT.write_text("\n".join(lines), encoding="utf-8")
    print(f"Ditulis: {OUT} ({len(soal)} soal)")


if __name__ == "__main__":
    main()
