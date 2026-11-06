// Package service (fitur reference) = daftar pilihan (kategori breakdown, status perbaikan, …).
// Semua user yang login boleh membaca; dipakai juga oleh input harian untuk memeriksa isian.
package service

import (
	"context"
	"strings"

	"simha/core-api/internal/reference/domain"
)

// Repository = kemampuan database yang dibutuhkan (dipenuhi *repository.Repository).
type Repository interface {
	List(ctx context.Context, group string) ([]domain.ReferenceValue, error)
}

type Service struct{ repo Repository }

func New(repo Repository) *Service { return &Service{repo: repo} }

// List = pilihan aktif satu kelompok (kosong = semua kelompok).
func (s *Service) List(ctx context.Context, group string) ([]domain.ReferenceValue, error) {
	return s.repo.List(ctx, strings.TrimSpace(group))
}
