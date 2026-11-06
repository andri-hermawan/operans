package service

import "testing"

// Tinggi kertas mengikuti tinggi isi supaya laporan selalu satu halaman, dalam batas yang wajar.
func TestPaperHeightInches(t *testing.T) {
	if h := paperHeightInches(100); h != minHeightIn {
		t.Errorf("isi pendek: tinggi %v, seharusnya A4 (%v)", h, minHeightIn)
	}
	// isi 2400 px (skala 1:1): 2400 ÷ 96 × 1,02 + 0,42 ≈ 25,9 inci → jauh lebih tinggi dari A4, tetapi satu halaman
	if h := paperHeightInches(2400); h < 25 || h > 27 {
		t.Errorf("isi 2400 px: tinggi %v", h)
	}
	// makin tinggi isi, makin tinggi kertas; tidak melewati batas Chrome
	if paperHeightInches(3000) <= paperHeightInches(2400) {
		t.Error("kertas harus makin tinggi bila isi makin tinggi")
	}
	if h := paperHeightInches(1e7); h != maxHeightIn {
		t.Errorf("batas atas: %v", h)
	}
}
