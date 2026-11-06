"""Penyimpanan file PDF di disk. Nama file = UUID (nama asli dari pengguna tidak pernah dipakai sebagai path)."""
from pathlib import Path
from uuid import UUID


class FileStore:
    def __init__(self, base_dir: str) -> None:
        self._base = Path(base_dir).resolve()

    @staticmethod
    def path_for(project_id: UUID, document_id: UUID) -> str:
        """Path relatif yang dicatat di ai.documents.storage_path."""
        return f"{project_id}/{document_id}.pdf"

    def save(self, rel: str, data: bytes) -> None:
        target = self._base / rel
        target.parent.mkdir(parents=True, exist_ok=True)
        target.write_bytes(data)

    def read(self, rel: str) -> bytes | None:
        """Isi file; None bila file tidak ada di penyimpanan."""
        target = self._base / rel
        return target.read_bytes() if target.is_file() else None

    def delete(self, rel: str) -> None:
        (self._base / rel).unlink(missing_ok=True)
