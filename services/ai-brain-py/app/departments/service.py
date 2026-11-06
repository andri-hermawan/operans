"""Logika fitur departments: semua role boleh melihat; hanya admin yang boleh menambah."""
from app.departments.domain import Department, DepartmentInput
from app.departments.repository import DepartmentRepository
from app.shared.errors import Forbidden
from app.shared.security import Principal


class DepartmentService:
    def __init__(self, repo: DepartmentRepository) -> None:
        self._repo = repo

    def list_departments(self, p: Principal) -> list[Department]:
        return self._repo.list_active(p.require_project())

    def create(self, p: Principal, data: DepartmentInput) -> Department:
        project_id = p.require_project()
        if not p.is_admin:
            raise Forbidden("hanya admin yang boleh menambah department")
        return self._repo.create(project_id, data)
