// Package domain (fitur project) = project tambang (halaman System Data → Project).
package domain

import (
	"time"

	"github.com/google/uuid"
)

// Project = data singkat project aktif (dipakai fitur lain: login, input harian, peta, …).
type Project struct {
	ID       uuid.UUID
	Code     string
	Name     string
	Timezone string
}

// ProjectDetail = project lengkap untuk halaman System Data → Project.
type ProjectDetail struct {
	ID          uuid.UUID `json:"id"`
	CompanyID   uuid.UUID `json:"company_id"`
	CompanyName string    `json:"company_name"`
	ProjectCode string    `json:"project_code"`
	ProjectName string    `json:"project_name"`
	Timezone    string    `json:"timezone"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type ProjectInput struct {
	CompanyID   *uuid.UUID `json:"company_id"`
	ProjectCode *string    `json:"project_code"`
	ProjectName *string    `json:"project_name"`
	Timezone    *string    `json:"timezone"`
	IsActive    *bool      `json:"is_active"`
}
