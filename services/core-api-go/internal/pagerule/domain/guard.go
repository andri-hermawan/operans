package domain

import "strings"

// Penjagaan di server (ADR 0018 tahap 2): endpoint data mana melayani halaman mana, dan aturan BAWAAN untuk halaman itu.
// Aturan bawaan harus sama dengan frontend (utils/permissions.ts): Report dan Record/Upload hanya untuk admin & dispatcher.

const dashboardKey = "dashboard"

// adminOnlyKeys = halaman yang bawaannya hanya admin dan tidak bisa dibukakan untuk role lain (setara resource "master").
var adminOnlyKeys = map[string]bool{"report/equipment-shift-performance": true}

// DefaultAllowed = aturan bawaan sebuah role untuk sebuah halaman (tanpa pengaturan superadmin).
func DefaultAllowed(role, key string) bool {
	switch {
	case adminOnlyKeys[key]:
		return role == "admin"
	case key == "report" || strings.HasPrefix(key, "report/") || strings.HasPrefix(key, "upload/"):
		return role == "admin" || role == "dispatcher"
	default:
		return true
	}
}

// Grantable = halaman yang boleh dibukakan lewat pengaturan untuk role yang bawaannya tertutup.
func Grantable(key string) bool { return !adminOnlyKeys[key] }

// Resolve = hak akhir satu role/user pada satu halaman: aturan user, lalu aturan role, lalu bawaan.
// Membukakan (true) hanya berlaku untuk halaman yang Grantable; memblokir (false) selalu berlaku.
func Resolve(role, key string, roleRules, userRules map[string]bool) bool {
	set, has := userRules[key]
	if !has {
		set, has = roleRules[key]
	}
	if !has {
		return DefaultAllowed(role, key)
	}
	if !set {
		return false
	}
	return Grantable(key) || DefaultAllowed(role, key)
}

// uploadPages = jenis data harian (bagian URL) → halaman Record / Upload yang memakainya.
var uploadPages = map[string]string{
	"operator-assignments":    "upload/daily-setting-operator",
	"equipment-activations":   "upload/equipment-activation",
	"equipment-breakdowns":    "upload/status-breakdown",
	"weighbridge-tickets":     "upload/weighbridge",
	"stock-emplacements":      "upload/stock-emplacement",
	"time-sheets":             "upload/time-sheet",
	"fuel-fillings":           "upload/fuel-filling",
	"ritases":                 "upload/ritase",
	"work-hour-notifications": "upload/wh-notification",
	"rain-slippery-logs":      "upload/rain-slippery",
	"yearly-plans":            "upload/plan-yearly",
	"monthly-plans":           "upload/plan-monthly",
}

// reportPages = laporan (bagian URL /reports/<nama>) → halaman yang memakainya. Rekap harian juga dipakai Dashboard
// (terbuka untuk semua role), jadi salah satu halaman terbuka sudah cukup.
var reportPages = map[string][]string{
	"executive-hauling":     {"report/report-executive-hauling"},
	"executive-emplacement": {"report/report-executive-emplacement"},
	"ritase-by-customer":    {"report/report-ritase-by-customer"},
	"weighbridge-daily": {
		dashboardKey, "report/report-ritase-daily", "report/report-payload-daily", "report/report-production-daily", "report/equipment-shift-performance",
	},
	"breakdown-daily": {dashboardKey, "report/report-breakdown-daily", "report/report-pa-daily", "report/equipment-shift-performance"},
}

// KeysFor = halaman yang memakai endpoint ini; user boleh bila SALAH SATU terbuka. nil = endpoint tidak dijaga.
// path = alamat permintaan, mis. "/api/v1/weighbridge-tickets/123" atau "/api/v1/imports/ritases/template".
func KeysFor(path string) []string {
	rest, ok := strings.CutPrefix(path, "/api/v1/")
	if !ok {
		return nil
	}
	seg := strings.Split(strings.Trim(rest, "/"), "/")
	switch seg[0] {
	case "imports", "reports":
		if len(seg) < 2 {
			return nil
		}
		if seg[0] == "reports" {
			return reportPages[seg[1]]
		}
		if k, ok := uploadPages[seg[1]]; ok {
			return []string{k}
		}
		return nil
	default:
		if seg[0] == "equipment-breakdowns" { // juga dibaca Report → Breakdown History
			return []string{"upload/status-breakdown", "report/report-breakdown-history"}
		}
		if seg[0] == "fuel-fillings" { // juga dibaca Report → Fuel Filling
			return []string{"upload/fuel-filling", "report/report-fuel-filling"}
		}
		if k, ok := uploadPages[seg[0]]; ok {
			return []string{k}
		}
		return nil
	}
}
