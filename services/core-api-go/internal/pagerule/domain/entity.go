// Package domain (fitur pagerule) = hak akses halaman & laporan yang diatur superadmin (ADR 0018).
// Hanya MENYIMPAN perubahan dari aturan bawaan; aturan bawaan sendiri ada di kode frontend (utils/permissions.ts).
package domain

import (
	"regexp"

	"github.com/google/uuid"
)

const (
	SubjectRole = "role"
	SubjectUser = "user"
)

// ProtectedKeys = halaman yang tidak boleh dikunci lewat aturan (agar tidak ada yang terkunci dari pengaturan hak akses).
var ProtectedKeys = map[string]bool{"master/access": true}

var keyRe = regexp.MustCompile(`^[a-z0-9][a-z0-9/_-]{0,78}$`)

// ValidKey = kunci halaman: huruf kecil, angka, "/", "-", "_" (mis. "report/report-ritase-daily").
func ValidKey(k string) bool { return keyRe.MatchString(k) }

func ValidRole(r string) bool { return r == "admin" || r == "dispatcher" || r == "viewer" }

// Rule = satu aturan tersimpan.
type Rule struct {
	SubjectType string     `json:"subject_type"`
	Role        *string    `json:"role,omitempty"`
	UserID      *uuid.UUID `json:"user_id,omitempty"`
	PageKey     string     `json:"page_key"`
	Allowed     bool       `json:"allowed"`
}

// Change = satu perubahan: Allowed nil = hapus aturan (kembali ke bawaan / ikut role).
type Change struct {
	PageKey string `json:"page_key"`
	Allowed *bool  `json:"allowed"`
}

// SaveInput = perubahan untuk SATU subjek (satu role atau satu user) di satu project.
type SaveInput struct {
	ProjectID   *uuid.UUID `json:"project_id"`
	SubjectType string     `json:"subject_type"`
	Role        string     `json:"role"`
	UserID      *uuid.UUID `json:"user_id"`
	Changes     []Change   `json:"changes"`
}

// Effective menggabungkan aturan untuk satu user: aturan role lebih dulu, aturan user menimpanya.
// Hasil hanya berisi halaman yang DIUBAH dari bawaan (kunci → boleh/tidak).
func Effective(roleRules, userRules []Rule) map[string]bool {
	out := make(map[string]bool, len(roleRules)+len(userRules))
	for _, r := range roleRules {
		out[r.PageKey] = r.Allowed
	}
	for _, r := range userRules {
		out[r.PageKey] = r.Allowed
	}
	return out
}
