"""Query SQL fitur departments (tabel ai.departments)."""
from uuid import UUID

from psycopg.errors import UniqueViolation
from psycopg.rows import dict_row

from app.departments.domain import Department, DepartmentInput
from app.shared.database import Database
from app.shared.errors import Conflict


class DepartmentRepository:
    def __init__(self, db: Database) -> None:
        self._db = db

    def list_active(self, project_id: UUID) -> list[Department]:
        with self._db.pool.connection() as conn, conn.cursor(row_factory=dict_row) as cur:
            cur.execute(
                "SELECT id, department_code, department_name FROM ai.departments"
                " WHERE project_id = %s AND is_active ORDER BY department_name",
                (project_id,),
            )
            return [Department(**r) for r in cur.fetchall()]

    def create(self, project_id: UUID, data: DepartmentInput) -> Department:
        try:
            with self._db.pool.connection() as conn, conn.cursor(row_factory=dict_row) as cur:
                cur.execute(
                    "INSERT INTO ai.departments (project_id, department_code, department_name)"
                    " VALUES (%s, %s, %s) RETURNING id, department_code, department_name",
                    (project_id, data.department_code.upper(), data.department_name),
                )
                return Department(**cur.fetchone())
        except UniqueViolation:
            raise Conflict("kode department sudah dipakai di project ini") from None
