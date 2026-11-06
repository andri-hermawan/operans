"""Waktu untuk tampilan (WIB) — dipakai tanda air dan label cetak."""
from datetime import datetime
from zoneinfo import ZoneInfo

WIB = ZoneInfo("Asia/Jakarta")


def now_wib_text() -> str:
    """Contoh: 04/11/2026 10:33 WIB."""
    return datetime.now(WIB).strftime("%d/%m/%Y %H:%M WIB")
