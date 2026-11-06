// import-fms memindahkan data CHR dari fms_lite (sistem lama) ke database simha (B8, ADR 0006/0008).
//
// fms_lite HANYA DIBACA: sambungan memakai default_transaction_read_only=on dan setiap query berjalan
// di transaksi READ ONLY. Aman dijalankan berulang (data yang sudah ada tidak digandakan; password user
// yang sudah dipindahkan tidak ditimpa).
//
// Data harian & peta melewati service yang sama dengan API (aturan validasi sama); baris yang tidak lolos
// dilaporkan dan dilewati, tidak dipaksa masuk.
//
// Env: FMS_DATABASE_URL (sumber, fms_lite) dan TARGET_DATABASE_URL (simha, role core_migrator).
// Biasanya dijalankan lewat tools\import-fms.ps1.
package main

import (
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"math"
	"os"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	activationrepo "simha/core-api/internal/activation/repository"
	activationsvc "simha/core-api/internal/activation/service"
	alertruledomain "simha/core-api/internal/alertrule/domain"
	alertrulerepo "simha/core-api/internal/alertrule/repository"
	alertrulesvc "simha/core-api/internal/alertrule/service"
	alerttypedomain "simha/core-api/internal/alerttype/domain"
	alerttyperepo "simha/core-api/internal/alerttype/repository"
	alerttypesvc "simha/core-api/internal/alerttype/service"
	breakdownrepo "simha/core-api/internal/breakdown/repository"
	breakdownsvc "simha/core-api/internal/breakdown/service"
	companyrepo "simha/core-api/internal/company/repository"
	companysvc "simha/core-api/internal/company/service"
	equipmentrepo "simha/core-api/internal/equipment/repository"
	equipmentsvc "simha/core-api/internal/equipment/service"
	fueldomain "simha/core-api/internal/fuelcalibration/domain"
	fuelcalibrationrepo "simha/core-api/internal/fuelcalibration/repository"
	fuelcalibrationsvc "simha/core-api/internal/fuelcalibration/service"
	importexcelsvc "simha/core-api/internal/importexcel/service"
	maparearepo "simha/core-api/internal/maparea/repository"
	mapareasvc "simha/core-api/internal/maparea/service"
	operatorrepo "simha/core-api/internal/operator/repository"
	operatorsvc "simha/core-api/internal/operator/service"
	projectrepo "simha/core-api/internal/project/repository"
	projectsvc "simha/core-api/internal/project/service"
	referencerepo "simha/core-api/internal/reference/repository"
	referencesvc "simha/core-api/internal/reference/service"
	"simha/core-api/internal/shared/access"
	"simha/core-api/internal/shared/dailyinput"
	"simha/core-api/internal/shared/database"
	shiftrepo "simha/core-api/internal/shift/repository"
	shiftsvc "simha/core-api/internal/shift/service"
	weighbridgerepo "simha/core-api/internal/weighbridge/repository"
	weighbridgesvc "simha/core-api/internal/weighbridge/service"
)

var (
	companyRMK = uuid.MustParse("5006fabf-3b40-4fa2-9260-c43b9f8452e0")
	projectCHR = uuid.MustParse("9811d93d-0813-489a-805d-eea4b68b3242")
)

const (
	superadminName = "Andri Hermawan" // satu-satunya superadmin (keputusan B-diskusi)
	importNote     = "impor fms_lite (B8)"
	chunkRows      = 4000
)

// project baru (keputusan B8): kode, nama, akun MCR yang menjadi admin-nya
var newProjects = []struct{ code, name, mcrUser string }{
	{"CMN", "Coal Mining", "MCR Mining"},
	{"EMP", "Emplacement", "MCR Emplasemen"},
}

type importer struct {
	src, dst   *pgxpool.Pool
	daily      *importexcelsvc.Service
	geo        *mapareasvc.Service
	fuel       *fuelcalibrationsvc.Service
	alertTypes *alerttypesvc.Service
	alertRules *alertrulesvc.Service
	actor      access.Actor
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "GAGAL:", err)
		os.Exit(1)
	}
}

func run() error {
	ctx := context.Background()
	srcURL, dstURL := os.Getenv("FMS_DATABASE_URL"), os.Getenv("TARGET_DATABASE_URL")
	if srcURL == "" || dstURL == "" {
		return errors.New("FMS_DATABASE_URL dan TARGET_DATABASE_URL wajib diisi")
	}
	srcCfg, err := pgxpool.ParseConfig(srcURL)
	if err != nil {
		return fmt.Errorf("FMS_DATABASE_URL: %w", err)
	}
	srcCfg.ConnConfig.RuntimeParams["default_transaction_read_only"] = "on" // lapis 1: sambungan baca-saja
	srcCfg.ConnConfig.RuntimeParams["application_name"] = "simha-import-readonly"
	src, err := pgxpool.NewWithConfig(ctx, srcCfg)
	if err != nil {
		return err
	}
	defer src.Close()
	dst, err := pgxpool.New(ctx, dstURL)
	if err != nil {
		return err
	}
	defer dst.Close()
	pool, err := database.Open(ctx, dstURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	im := newImporter(src, dst, pool)
	steps := []struct {
		name string
		fn   func(context.Context) error
	}{
		{"1. Perusahaan, project & shift", im.projects},
		{"2. User & keanggotaan", im.users},
		{"3. Equipment", im.equipment},
		{"4. Aktivasi equipment", im.activations},
		{"5. Operator harian", im.operators},
		{"6. Breakdown", im.breakdowns},
		{"7. Weighbridge", im.weighbridge},
		{"8. Peta CHR", im.mapCHR},
		{"9. Kalibrasi fuel", im.calibrations},
		{"10. Penerima alert", im.alertRecipients},
	}
	for _, s := range steps {
		fmt.Printf("\n== %s\n", s.name)
		if err := s.fn(ctx); err != nil {
			return fmt.Errorf("%s: %w", s.name, err)
		}
	}
	fmt.Println("\nSELESAI.")
	return nil
}

// read menjalankan query sumber di transaksi READ ONLY (lapis 2) dan memanggil fn untuk tiap baris.
func (im *importer) read(ctx context.Context, sql string, fn func(pgx.Rows) error, args ...any) error {
	tx, err := im.src.BeginTx(ctx, pgx.TxOptions{AccessMode: pgx.ReadOnly})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	rows, err := tx.Query(ctx, sql, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		if err := fn(rows); err != nil {
			return err
		}
	}
	return rows.Err()
}

// ---------------------------------------------------------------------
// 1–3. Data induk (SQL langsung, satu transaksi per langkah)
// ---------------------------------------------------------------------

func (im *importer) projects(ctx context.Context) error {
	tx, err := im.dst.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `
		INSERT INTO core.companies (id, company_code, company_name) VALUES ($1, 'RMK', 'PT. RMK')
		ON CONFLICT DO NOTHING`, companyRMK); err != nil {
		return err
	}
	for _, p := range newProjects {
		tag, err := tx.Exec(ctx, `
			INSERT INTO core.projects (company_id, project_code, project_name, timezone)
			VALUES ($1, $2, $3, 'Asia/Jakarta') ON CONFLICT (project_code) DO NOTHING`, companyRMK, p.code, p.name)
		if err != nil {
			return err
		}
		// shift sama dengan CHR (07:00–19:00, 19:00–07:00)
		if _, err := tx.Exec(ctx, `
			INSERT INTO core.shifts (project_id, shift_code, shift_name, start_time, end_time, sequence)
			SELECT p.id, s.shift_code, s.shift_name, s.start_time, s.end_time, s.sequence
			  FROM core.projects p, core.shifts s
			 WHERE p.project_code = $1 AND s.project_id = $2
			ON CONFLICT DO NOTHING`, p.code, projectCHR); err != nil {
			return err
		}
		fmt.Printf("   project %s (%s): %s\n", p.code, p.name, created(tag.RowsAffected()))
	}
	return tx.Commit(ctx)
}

func (im *importer) projectID(ctx context.Context, code string) (uuid.UUID, error) {
	var id uuid.UUID
	err := im.dst.QueryRow(ctx, `SELECT id FROM core.projects WHERE project_code = $1`, code).Scan(&id)
	return id, err
}

type legacyUser struct {
	id                       uuid.UUID
	name, email, role, phone string
	hash                     string
	active                   bool
}

func (im *importer) users(ctx context.Context) error {
	var users []legacyUser
	if err := im.read(ctx, `
		SELECT id, name, email, password_hash, role, status, coalesce(phone_number, '')
		  FROM users WHERE project_id = $1 ORDER BY name`, func(r pgx.Rows) error {
		var u legacyUser
		var status string
		if err := r.Scan(&u.id, &u.name, &u.email, &u.hash, &u.role, &status, &u.phone); err != nil {
			return err
		}
		u.active = status == "active"
		users = append(users, u)
		return nil
	}, projectCHR); err != nil {
		return err
	}

	extra := map[string]uuid.UUID{} // nama akun MCR → project baru
	for _, p := range newProjects {
		id, err := im.projectID(ctx, p.code)
		if err != nil {
			return err
		}
		extra[p.mcrUser] = id
	}

	tx, err := im.dst.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var nNew, nOld, nSuper int
	for _, u := range users {
		isSuper := u.name == superadminName
		if isSuper {
			nSuper++
			im.actor = access.Actor{UserID: u.id, SuperAdmin: true}
		}
		tag, err := tx.Exec(ctx, `
			INSERT INTO auth.users (id, email, name, phone_number, password_hash, is_superadmin, is_active)
			VALUES ($1, lower(btrim($2)), $3, $4, $5, $6, $7) ON CONFLICT DO NOTHING`,
			u.id, u.email, u.name, phone(u.phone), u.hash, isSuper, u.active)
		if err != nil {
			return fmt.Errorf("user %s: %w", u.name, err)
		}
		if tag.RowsAffected() == 1 {
			nNew++
		} else {
			nOld++
		}
		// semua anggota CHR; superadmin lama → admin; MCR Emplasemen/Mining juga admin project barunya
		role := map[string]string{"superadmin": "admin", "admin": "admin", "viewer": "viewer", "dispatcher": "dispatcher"}[u.role]
		if role == "" {
			role = "viewer"
		}
		memberships := map[uuid.UUID]string{projectCHR: role}
		if pid, ok := extra[u.name]; ok {
			memberships[pid] = "admin"
		}
		for pid, r := range memberships {
			if _, err := tx.Exec(ctx, `
				INSERT INTO auth.project_members (user_id, project_id, role) VALUES ($1, $2, $3)
				ON CONFLICT DO NOTHING`, u.id, pid, r); err != nil {
				return err
			}
		}
		fmt.Printf("   %-20s lama=%-10s → %s\n", u.name, u.role, describe(isSuper, role, extra[u.name] != uuid.Nil))
	}
	if nSuper != 1 {
		return fmt.Errorf("superadmin %q tidak ditemukan tepat satu (ketemu %d)", superadminName, nSuper)
	}
	fmt.Printf("   user baru %d, sudah ada %d (password yang sudah ada tidak ditimpa)\n", nNew, nOld)
	return tx.Commit(ctx)
}

func describe(super bool, role string, extra bool) string {
	s := role + " CHR"
	if super {
		s = "SUPERADMIN + " + s
	}
	if extra {
		s += " + admin project baru"
	}
	return s
}

// phone: format sama dengan API (angka, boleh diawali +, 8–20 karakter); salah → dikosongkan.
func phone(p string) *string {
	s := strings.NewReplacer(" ", "", "-", "").Replace(strings.TrimSpace(p))
	if len(s) < 8 || len(s) > 20 || strings.Trim(strings.TrimPrefix(s, "+"), "0123456789") != "" {
		return nil
	}
	return &s
}

func (im *importer) equipment(ctx context.Context) error {
	tx, err := im.dst.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	n := 0
	err = im.read(ctx, `
		SELECT id, equipment_code, equipment_alias, type, brand, model, class, status
		  FROM equipments WHERE project_id = $1 ORDER BY equipment_code`, func(r pgx.Rows) error {
		var id uuid.UUID
		var code string
		var alias, typ, brand, model, class *string
		var status string
		if err := r.Scan(&id, &code, &alias, &typ, &brand, &model, &class, &status); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `
			INSERT INTO core.equipment (id, project_id, equipment_code, equipment_alias, equipment_type, brand, model,
			                            class, is_active)
			VALUES ($1, $2, upper(btrim($3)), $4, $5, $6, $7, $8, $9)
			ON CONFLICT (id) DO UPDATE SET equipment_code = EXCLUDED.equipment_code,
			       equipment_alias = EXCLUDED.equipment_alias, equipment_type = EXCLUDED.equipment_type,
			       brand = EXCLUDED.brand, model = EXCLUDED.model, class = EXCLUDED.class, is_active = EXCLUDED.is_active`,
			id, projectCHR, code, alias, typ, brand, model, class, status == "active")
		n++
		fmt.Printf("   %s (%s)\n", code, id)
		return err
	}, projectCHR)
	if err != nil {
		return err
	}
	fmt.Printf("   %d equipment (UUID sama dengan fms_lite)\n", n)
	return tx.Commit(ctx)
}

// ---------------------------------------------------------------------
// 4–7. Data harian: lewat impor CSV service (aturan sama dengan upload Excel)
// ---------------------------------------------------------------------

func (im *importer) activations(ctx context.Context) error {
	var rows [][]string
	if err := im.read(ctx, `
		SELECT to_char(date_at, 'YYYY-MM-DD'), equipment_code, status, coalesce(description, '')
		  FROM equipment_activation ORDER BY date_at, id`, collect(&rows, 4)); err != nil {
		return err
	}
	return im.importCSV(ctx, "equipment-activations", []string{"Date", "Equipment Code", "Status", "Description"}, rows)
}

func (im *importer) operators(ctx context.Context) error {
	var rows [][]string
	if err := im.read(ctx, `
		SELECT to_char(date_at, 'YYYY-MM-DD'), shift, equipment_code, operator_name, coalesce(description, '')
		  FROM daily_setting_operator ORDER BY date_at, id`, collect(&rows, 5)); err != nil {
		return err
	}
	return im.importCSV(ctx, "operator-assignments", []string{"Tanggal", "Shift", "Asset ID", "Operator Name", "Description"}, rows)
}

func (im *importer) breakdowns(ctx context.Context) error {
	var rows [][]string
	// durasi lama = jam desimal; hanya dipakai bila jam selesai kosong (bila ada, durasi dihitung ulang)
	if err := im.read(ctx, `
		SELECT to_char(date_at, 'YYYY-MM-DD'), shift, equipment_code, '', status, coalesce(category, ''),
		       coalesce(to_char(time_start, 'HH24:MI'), ''), coalesce(to_char(time_end, 'HH24:MI'), ''),
		       CASE WHEN time_end IS NULL AND duration IS NOT NULL
		            THEN floor(duration)::int || ':' || lpad(round((duration - floor(duration)) * 60)::int::text, 2, '0')
		            ELSE '' END,
		       coalesce(repair_status, ''), coalesce(description, ''), coalesce(location, '')
		  FROM breakdown_status ORDER BY date_at, id`, collect(&rows, 12)); err != nil {
		return err
	}
	return im.importCSV(ctx, "equipment-breakdowns", []string{"Date", "Shift", "Asset ID", "Class", "Status", "Category",
		"Time Start", "Time End", "Duration", "Repair Status", "Description", "Location"}, rows)
}

func (im *importer) weighbridge(ctx context.Context) error {
	var rows [][]string
	if err := im.read(ctx, `
		SELECT to_char(date_at, 'YYYY-MM-DD'), shift, ticket_no, equipment_code, coalesce(product, ''),
		       coalesce(gross::text, ''), coalesce(tare::text, ''), coalesce(net::text, ''),
		       coalesce(recipient, ''), coalesce(customer, ''), coalesce(transporter, ''),
		       coalesce(to_char(gross_time, 'HH24:MI'), ''), coalesce(to_char(tare_time, 'HH24:MI'), ''),
		       coalesce(gross_operator, ''), coalesce(tare_operator, ''), coalesce(description, ''), coalesce(location, '')
		  FROM weighbridge ORDER BY date_at, id`, collect(&rows, 17)); err != nil {
		return err
	}
	return im.importCSV(ctx, "weighbridge-tickets", []string{"Date", "Shift", "Ticket No", "Asset ID", "Product", "Gross",
		"Tare", "Net", "Recipient", "Customer", "Transporter", "Gross Time", "Tare Time", "Gross Operator",
		"Tare Operator", "Description", "Location"}, rows)
}

func collect(dst *[][]string, n int) func(pgx.Rows) error {
	return func(r pgx.Rows) error {
		vals := make([]string, n)
		ptrs := make([]any, n)
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := r.Scan(ptrs...); err != nil {
			return err
		}
		*dst = append(*dst, vals)
		return nil
	}
}

// importCSV: per potongan, pratinjau dulu → buang baris yang salah (dilaporkan) → simpan sisanya.
func (im *importer) importCSV(ctx context.Context, typ string, header []string, rows [][]string) error {
	var created, updated, dup int
	var rejectedRows [][]string
	reasons := map[string]int{}
	var examples []string
	for start := 0; start < len(rows); start += chunkRows {
		chunk := rows[start:min(start+chunkRows, len(rows))]
		res, err := im.daily.Import(ctx, im.actor, typ, &projectCHR, "fms.csv", toCSV(header, chunk), true)
		if err != nil {
			return err
		}
		bad := map[int]bool{}
		if len(res.Errors) >= 500 { // daftar kesalahan dibatasi service; potongan harus diperkecil
			return fmt.Errorf("terlalu banyak baris salah dalam satu potongan (%d)", res.InvalidRows)
		}
		for _, e := range res.Errors {
			bad[e.Row] = true
			rejectedRows = append(rejectedRows, append([]string{e.Column + ": " + e.Message}, chunk[e.Row-2]...))
			key := e.Column + ": " + firstWords(e.Message)
			reasons[key]++
			if len(examples) < 5 {
				examples = append(examples, fmt.Sprintf("%s %s | %s: %s", chunk[e.Row-2][0], chunk[e.Row-2][2], e.Column, e.Message))
			}
		}
		var keep [][]string
		for i, r := range chunk {
			if !bad[i+2] {
				keep = append(keep, r)
			}
		}
		if len(keep) == 0 {
			continue
		}
		res, err = im.daily.Import(ctx, im.actor, typ, &projectCHR, "fms.csv", toCSV(header, keep), false)
		if err != nil {
			return err
		}
		created, updated, dup = created+res.ToCreate, updated+res.ToUpdate, dup+res.DuplicateRows
	}
	rejected := 0
	for _, n := range reasons {
		rejected += n
	}
	if err := writeRejected(typ, header, rejectedRows); err != nil {
		return err
	}
	fmt.Printf("   baris lama %d → baru %d, diperbarui %d, kunci ganda di data lama %d, DITOLAK %d\n",
		len(rows), created, updated, dup, rejected)
	keys := make([]string, 0, len(reasons))
	for k := range reasons {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return reasons[keys[i]] > reasons[keys[j]] })
	for _, k := range keys {
		fmt.Printf("     - %4d × %s\n", reasons[k], k)
	}
	for _, e := range examples {
		fmt.Printf("       contoh: %s\n", e)
	}
	return nil
}

// writeRejected menyimpan baris yang ditolak ke laporan-impor/ditolak_<jenis>.csv (kolom pertama = alasan),
// agar bisa diperiksa/diperbaiki lalu diunggah lewat menu impor Excel. Tanpa baris ditolak → file dihapus.
func writeRejected(typ string, header []string, rows [][]string) error {
	path := "laporan-impor/ditolak_" + typ + ".csv"
	if len(rows) == 0 {
		_ = os.Remove(path)
		return nil
	}
	if err := os.MkdirAll("laporan-impor", 0o755); err != nil {
		return err
	}
	var b bytes.Buffer
	b.WriteString("\xEF\xBB\xBF") // BOM agar Excel membaca UTF-8
	w := csv.NewWriter(&b)
	_ = w.Write(append([]string{"Alasan ditolak"}, header...))
	_ = w.WriteAll(rows)
	fmt.Printf("     laporan: %s\n", path)
	return os.WriteFile(path, b.Bytes(), 0o644)
}

func toCSV(header []string, rows [][]string) *bytes.Reader {
	var b bytes.Buffer
	w := csv.NewWriter(&b)
	_ = w.Write(header)
	_ = w.WriteAll(rows)
	return bytes.NewReader(b.Bytes())
}

// firstWords meringkas pesan kesalahan agar bisa dikelompokkan: angka & nilai dikutip diganti "…".
func firstWords(msg string) string {
	var b strings.Builder
	skip := false
	for _, r := range msg {
		isVal := r == '"' || (r >= '0' && r <= '9') || ((r == ':' || r == '.' || r == '–') && skip)
		if isVal && !skip {
			b.WriteString("…")
		}
		if !isVal {
			b.WriteRune(r)
		}
		skip = isVal
	}
	return strings.TrimSpace(b.String())
}

// ---------------------------------------------------------------------
// 8–10. Peta, kalibrasi, penerima alert (lewat service)
// ---------------------------------------------------------------------

func (im *importer) mapCHR(ctx context.Context) error {
	list, err := im.geo.Imports(ctx, im.actor, &projectCHR)
	if err != nil {
		return err
	}
	for _, l := range list {
		if l.IsCurrent {
			fmt.Printf("   dilewati: CHR sudah punya peta aktif (%s, %d area)\n", l.FileName, l.FeatureCount)
			return nil
		}
	}
	var raw string
	if err := im.read(ctx, `SELECT geojson_origin::text FROM projects WHERE id = $1`, func(r pgx.Rows) error {
		return r.Scan(&raw)
	}, projectCHR); err != nil {
		return err
	}
	note := importNote
	res, err := im.geo.Import(ctx, im.actor, &projectCHR, "peta_chr_fms_lite.geojson", &note, strings.NewReader(raw), false)
	if err != nil {
		return err
	}
	fmt.Printf("   %d area, %d kategori, %d geometri diperbaiki, %d peringatan\n",
		res.ValidFeatures, len(res.Categories), res.FixedGeometries, len(res.Warnings))
	return nil
}

func (im *importer) calibrations(ctx context.Context) error {
	type head struct {
		typ    string
		points []fueldomain.CalibrationPoint
	}
	byEq := map[uuid.UUID]*head{}
	var order []uuid.UUID
	if err := im.read(ctx, `
		SELECT equipment_id, lower(coalesce(calibration_type, 'LLS')), fuel_level::float8, fuel_volume::float8
		  FROM fuel_calibrations ORDER BY equipment_id, fuel_level`, func(r pgx.Rows) error {
		var id uuid.UUID
		var typ string
		var p fueldomain.CalibrationPoint
		if err := r.Scan(&id, &typ, &p.SensorValue, &p.VolumeLiters); err != nil {
			return err
		}
		if byEq[id] == nil {
			byEq[id] = &head{typ: typ}
			order = append(order, id)
		}
		byEq[id].points = append(byEq[id].points, p)
		return nil
	}); err != nil {
		return err
	}
	for _, id := range order {
		h := byEq[id]
		last := h.points[len(h.points)-1]
		in := fuelcalibrationsvc.FuelCalibrationInput{CalibrationType: h.typ}
		if samePoints(fueldomain.LinearPoints(last.VolumeLiters, last.SensorValue, h.typ), h.points) {
			in.TankCapacityLiters, in.MaxSensorValue = &last.VolumeLiters, &last.SensorValue // garis lurus seperti lama
		} else {
			in.Points = h.points
		}
		c, _, err := im.fuel.Set(ctx, im.actor, id, in)
		if err != nil {
			return fmt.Errorf("equipment %s: %w", id, err)
		}
		fmt.Printf("   %s: %s %v L / sensor maks %v, %d titik, linear=%v\n",
			c.EquipmentCode, c.CalibrationType, c.TankCapacityLiters, c.MaxSensorValue, c.PointCount, c.IsLinear)
	}
	return nil
}

func samePoints(a, b []fueldomain.CalibrationPoint) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if math.Abs(a[i].SensorValue-b[i].SensorValue) > 0.005 || math.Abs(a[i].VolumeLiters-b[i].VolumeLiters) > 0.005 {
			return false
		}
	}
	return true
}

func (im *importer) alertRecipients(ctx context.Context) error {
	types, err := im.alertTypes.List(ctx)
	if err != nil {
		return err
	}
	byLegacy := map[string]alerttypedomain.AlertType{}
	for _, t := range types {
		byLegacy[t.LegacyCode] = t
	}
	// ambang global lama dibandingkan dengan nilai bawaan jenis alert (seharusnya sama)
	recipients := map[string][]uuid.UUID{}
	if err := im.read(ctx, `
		SELECT c.alert_category_code, r.duration_minutes::float8, r.speed_limit::float8, r.fuel_threshold::float8,
		       coalesce(array_agg(x.user_id) FILTER (WHERE x.user_id IS NOT NULL), '{}')
		  FROM alert_rules r
		  JOIN alert_categories c ON c.id = r.alert_category_id
		  LEFT JOIN alert_rule_recipients x ON x.alert_rule_id = r.id
		 WHERE r.status = 'active' AND (r.project_id IS NULL OR r.project_id = $1)
		 GROUP BY c.alert_category_code, r.duration_minutes, r.speed_limit, r.fuel_threshold`, func(r pgx.Rows) error {
		var code string
		var dur, speed, fuel *float64
		var users []uuid.UUID
		if err := r.Scan(&code, &dur, &speed, &fuel, &users); err != nil {
			return err
		}
		t, ok := byLegacy[code]
		if !ok {
			fmt.Printf("   ! kode alert lama %s tidak dikenal, dilewati\n", code)
			return nil
		}
		if fuel != nil {
			v := math.Abs(*fuel)
			fuel = &v
		}
		if !eq(dur, t.DefaultDurationMinutes) || !eq(speed, t.DefaultSpeedLimitKph) || !eq(fuel, t.DefaultFuelDropLiters) {
			fmt.Printf("   ! ambang lama %s berbeda dengan bawaan baru (durasi %v, speed %v, fuel %v)\n", t.Code, val(dur), val(speed), val(fuel))
		}
		recipients[t.Code] = append(recipients[t.Code], users...)
		return nil
	}, projectCHR); err != nil {
		return err
	}

	existing, err := im.alertRules.List(ctx, im.actor, &projectCHR)
	if err != nil {
		return err
	}
	for _, t := range types {
		ids := recipients[t.Code]
		if len(ids) == 0 {
			continue
		}
		in := alertrulesvc.AlertRuleInput{ProjectID: &projectCHR, AlertType: t.Code, RecipientUserIDs: ids,
			Description: ptr("penerima dari fms_lite (B8); ambang ikut bawaan")}
		var rule *alertruledomain.AlertRule
		var warnings []string
		cur := findRule(existing, t.Code)
		if cur != nil { // pertahankan ambang yang sudah diatur, ganti penerimanya
			in.DurationMinutes, in.SpeedLimitKph, in.FuelDropLiters = cur.DurationMinutes, cur.SpeedLimitKph, cur.FuelDropLiters
			in.Description, in.IsActive = cur.Description, &cur.IsActive
			rule, warnings, err = im.alertRules.Replace(ctx, im.actor, cur.ID, in)
		} else {
			rule, warnings, err = im.alertRules.Create(ctx, im.actor, in)
		}
		if err != nil {
			return fmt.Errorf("%s: %w", t.Code, err)
		}
		names := make([]string, len(rule.Recipients))
		for i, r := range rule.Recipients {
			names[i] = r.Name
		}
		fmt.Printf("   %-26s %d penerima: %s\n", t.Code, len(names), strings.Join(names, ", "))
		for _, w := range warnings {
			fmt.Printf("     peringatan: %s\n", w)
		}
	}
	return nil
}

func findRule(rules []alertruledomain.AlertRule, typ string) *alertruledomain.AlertRule {
	for i := range rules {
		if rules[i].AlertType == typ && rules[i].ZoneName == nil {
			return &rules[i]
		}
	}
	return nil
}

func eq(a, b *float64) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return math.Abs(*a-*b) < 0.005
}

func val(p *float64) any {
	if p == nil {
		return "-"
	}
	return *p
}

func ptr(s string) *string { return &s }

func created(n int64) string {
	if n == 1 {
		return "dibuat"
	}
	return "sudah ada"
}

// newImporter merakit service yang dipakai alat ini dengan cara yang SAMA seperti cmd/api/main.go
// (data harian & peta melewati aturan validasi yang sama dengan API).
func newImporter(src, dst, pool *pgxpool.Pool) *importer {
	projects := projectsvc.New(projectrepo.New(pool), companysvc.New(companyrepo.New(pool)))
	shifts := shiftsvc.New(shiftrepo.New(pool), projects)
	equipment := equipmentsvc.New(equipmentrepo.New(pool), projects)
	daily := dailyinput.Loader{Projects: projects, Shifts: shifts, Equipment: equipment,
		References: referencesvc.New(referencerepo.New(pool))}
	alertTypes := alerttypesvc.New(alerttyperepo.New(pool))
	mapArea := mapareasvc.New(maparearepo.New(pool), projects)
	return &importer{
		src: src, dst: dst,
		daily: importexcelsvc.New(daily, map[string]importexcelsvc.RowImporter{
			"operator-assignments":  operatorsvc.New(operatorrepo.New(pool), daily),
			"equipment-activations": activationsvc.New(activationrepo.New(pool), equipment),
			"equipment-breakdowns":  breakdownsvc.New(breakdownrepo.New(pool), daily),
			"weighbridge-tickets":   weighbridgesvc.New(weighbridgerepo.New(pool), daily),
		}),
		geo:        mapArea,
		fuel:       fuelcalibrationsvc.New(fuelcalibrationrepo.New(pool), equipment),
		alertTypes: alertTypes,
		alertRules: alertrulesvc.New(alertrulerepo.New(pool), projects, alertTypes, mapArea),
	}
}
