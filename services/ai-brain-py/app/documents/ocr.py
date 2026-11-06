"""OCR untuk halaman PDF tanpa teks (hasil scan). OPSIONAL: aktif bila OCR_ENABLED=true dan pustakanya terpasang
(`pip install -e ".[ocr]"`). Mesin: RapidOCR (onnxruntime, berjalan di CPU, tanpa program tambahan) dengan model
huruf Latin PP-OCRv5 — model bawaan lama (Mandarin/Inggris) sering menggabungkan kata ("operatorwajibmemakai…"),
yang membuat hasilnya tidak bisa dicari kata per kata.

Model diunduh otomatis saat OCR pertama kali dipakai (butuh internet sekali), lalu tersimpan di folder paket rapidocr.
Di server tanpa internet: jalankan sekali di mesin ber-internet, lalu salin folder `rapidocr/models`.
"""
import logging
import threading
from typing import Protocol

log = logging.getLogger("ai-brain")


class OcrEngine(Protocol):
    available: bool

    def read(self, pdf_bytes: bytes, page_indexes: list[int]) -> dict[int, str]:
        """Teks hasil OCR per indeks halaman (mulai 0)."""
        ...


class NoOcr:
    """OCR tidak aktif: halaman tanpa teks dibiarkan kosong."""

    available = False

    def read(self, pdf_bytes: bytes, page_indexes: list[int]) -> dict[int, str]:
        return {}


class RapidOcrEngine:
    available = True
    _MIN_CONFIDENCE = 0.5
    _RENDER_SCALE = 2.0  # ≈ 144 dpi: huruf kecil tetap terbaca, waktu proses masih wajar

    def __init__(self) -> None:
        self._engine = None
        self._lock = threading.Lock()  # satu OCR pada satu waktu: berat di CPU dan mesinnya tidak dijamin aman-thread

    def _load(self):
        from rapidocr import LangRec, ModelType, OCRVersion, RapidOCR

        return RapidOCR(params={
            "Rec.lang_type": LangRec.LATIN,  # huruf Latin: Indonesia/Inggris
            "Rec.ocr_version": OCRVersion.PPOCRV5,
            "Rec.model_type": ModelType.MOBILE,
        })

    def read(self, pdf_bytes: bytes, page_indexes: list[int]) -> dict[int, str]:
        import numpy as np
        import pypdfium2 as pdfium

        with self._lock:
            if self._engine is None:
                self._engine = self._load()
            pdf = pdfium.PdfDocument(pdf_bytes)
            try:
                out: dict[int, str] = {}
                for i in page_indexes:
                    image = pdf[i].render(scale=self._RENDER_SCALE).to_pil().convert("RGB")
                    result = self._engine(np.array(image))
                    texts = list(result.txts or [])
                    scores = list(result.scores or [])
                    out[i] = "\n".join(t for t, s in zip(texts, scores) if float(s) >= self._MIN_CONFIDENCE)
                return out
            finally:
                pdf.close()


def build_ocr(enabled: bool) -> OcrEngine:
    if not enabled:
        return NoOcr()
    try:
        import rapidocr  # noqa: F401
    except ImportError:
        log.warning("OCR_ENABLED=true tetapi rapidocr belum terpasang (pip install -e \".[ocr]\"); OCR dimatikan")
        return NoOcr()
    return RapidOcrEngine()
