"""Bentuk data fitur departments."""
from uuid import UUID

from pydantic import BaseModel, Field


class Department(BaseModel):
    id: UUID
    department_code: str
    department_name: str


class DepartmentInput(BaseModel):
    department_code: str = Field(min_length=1, max_length=30, pattern=r"^[A-Za-z0-9_-]+$")
    department_name: str = Field(min_length=1, max_length=100)
