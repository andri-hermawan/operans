// Package domain (fitur company) = perusahaan pemilik project (halaman System Data → Company).
package domain

import (
	"time"

	"github.com/google/uuid"
)

// Company = perusahaan pemilik project (halaman System Data → Company).
type Company struct {
	ID           uuid.UUID `json:"id"`
	CompanyCode  string    `json:"company_code"`
	CompanyName  string    `json:"company_name"`
	IsActive     bool      `json:"is_active"`
	ProjectCount int       `json:"project_count"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type CompanyInput struct {
	CompanyCode *string `json:"company_code"`
	CompanyName *string `json:"company_name"`
	IsActive    *bool   `json:"is_active"`
}
