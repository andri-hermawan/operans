package service

import (
	"context"
	"fmt"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

// Ukuran halaman PDF mengikuti laporan di layar: SATU halaman selebar 1280 px (13,33 inci), skala asli 1:1 (tulisan tidak
// dikecilkan), dan tinggi mengikuti isi sehingga tidak ada grafik/data yang terpotong. Isi laporan dikunci selebar
// 1240 px (lihat halaman cetak) dan tata letak dua kolom tidak bergantung breakpoint layar.
const (
	printScale   = 1.0
	pageWidthPx  = 1280.0
	pageMarginPx = 20.0
	paperWidthIn = pageWidthPx / 96
	marginIn     = pageMarginPx / 96
	minHeightIn  = 8.27  // kertas tidak pernah lebih pendek dari ini
	maxHeightIn  = 190.0 // batas Chrome untuk satu halaman PDF ±200 inci
	heightSafety = 1.02  // cadangan 2% agar tidak ada baris yang tergeser ke halaman kedua
)

// paperHeightInches = tinggi kertas (inci) supaya SELURUH laporan muat dalam SATU halaman: tinggi isi (px CSS) × skala ÷ 96,
// ditambah margin atas-bawah. Laporan dengan banyak zona memang jadi halaman tinggi, tetapi tidak ada grafik/data yang terpotong.
func paperHeightInches(contentPx float64) float64 {
	h := contentPx*printScale/96*heightSafety + 2*marginIn + 0.1
	if h < minHeightIn {
		return minHeightIn
	}
	if h > maxHeightIn {
		return maxHeightIn
	}
	return h
}

// renderPDF membuka url di Chrome/Edge tanpa layar, menunggu window.__REPORT_READY__ === true, lalu
// mencetak SATU halaman selebar 1280 px dengan tinggi menyesuaikan isi. browserPath kosong = Chrome/Edge dicari otomatis.
func renderPDF(ctx context.Context, browserPath, url string) ([]byte, error) {
	opts := append(chromedp.DefaultExecAllocatorOptions[:], chromedp.WindowSize(int(pageWidthPx), 1000))
	if browserPath != "" {
		opts = append(opts, chromedp.ExecPath(browserPath))
	}
	alloc, cancelAlloc := chromedp.NewExecAllocator(ctx, opts...)
	defer cancelAlloc()
	tab, cancelTab := chromedp.NewContext(alloc)
	defer cancelTab()
	tab, cancel := context.WithTimeout(tab, 2*time.Minute)
	defer cancel()

	var ready bool
	var pdf []byte
	err := chromedp.Run(tab,
		chromedp.Navigate(url),
		chromedp.Poll(`window.__REPORT_READY__ === true || (window.__REPORT_ERROR__ ? true : false)`, &ready, chromedp.WithPollingTimeout(90*time.Second)),
		chromedp.ActionFunc(func(ctx context.Context) error {
			var msg string
			if err := chromedp.Evaluate(`window.__REPORT_ERROR__ || ""`, &msg).Do(ctx); err == nil && msg != "" {
				return fmt.Errorf("halaman cetak: %s", msg)
			}
			// ukur tinggi isi setelah grafik tergambar, lalu sesuaikan tinggi kertas agar semuanya muat di satu halaman
			var contentPx float64
			if err := chromedp.Evaluate(`Math.max(document.documentElement.scrollHeight, document.body.scrollHeight)`, &contentPx).Do(ctx); err != nil {
				return fmt.Errorf("ukur tinggi halaman: %w", err)
			}
			var err error
			pdf, _, err = page.PrintToPDF().WithPrintBackground(true).WithScale(printScale).
				WithPaperWidth(paperWidthIn).WithPaperHeight(paperHeightInches(contentPx)).WithMarginTop(marginIn).WithMarginBottom(marginIn).
				WithMarginLeft(marginIn).WithMarginRight(marginIn).Do(ctx)
			return err
		}),
	)
	return pdf, err
}
