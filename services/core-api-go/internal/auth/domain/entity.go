// Package domain (fitur auth) = data login: akun user (termasuk hash password & kunci akun)
// dan refresh token sesi.
package domain

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID               uuid.UUID
	Email            string
	Name             string
	PhoneNumber      *string
	PasswordHash     string
	IsSuperadmin     bool
	IsActive         bool
	FailedLoginCount int
	LockedUntil      *time.Time
	LastLoginAt      *time.Time
}

// RefreshToken = satu refresh token di database (hanya hash, bukan token asli).
type RefreshToken struct {
	ID                uuid.UUID
	FamilyID          uuid.UUID // satu sesi login = satu keluarga token
	UserID            uuid.UUID
	ProjectID         *uuid.UUID
	TokenHash         string
	IssuedAt          time.Time
	ExpiresAt         time.Time // habis bila tidak dipakai (sliding)
	AbsoluteExpiresAt time.Time // batas mutlak sejak login
	UsedAt            *time.Time
	RevokedAt         *time.Time
	UserAgent         string
	IPAddress         string
}
