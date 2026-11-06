// Package domain (fitur user) = data halaman kelola user: ringkasan user + keanggotaan project.
package domain

import (
	"time"

	"github.com/google/uuid"

	"simha/core-api/internal/shared/access"
)

// UserSummary = data user yang ditampilkan di halaman kelola user (TANPA hash password).
type UserSummary struct {
	ID           uuid.UUID           `json:"id"`
	Email        string              `json:"email"`
	Name         string              `json:"name"`
	PhoneNumber  *string             `json:"phone_number"`
	IsSuperadmin bool                `json:"is_superadmin"`
	IsActive     bool                `json:"is_active"`
	LastLoginAt  *time.Time          `json:"last_login_at"`
	CreatedAt    time.Time           `json:"created_at"`
	Memberships  []access.Membership `json:"memberships"`
}

// UserUpdate = perubahan sebagian; nil = tidak diubah.
type UserUpdate struct {
	Name         *string `json:"name"`
	PhoneNumber  *string `json:"phone_number"` // "" = dihapus
	IsActive     *bool   `json:"is_active"`
	IsSuperadmin *bool   `json:"is_superadmin"`
}
