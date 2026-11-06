// create-user: membuat user langsung di database (sebelum halaman kelola user di B3 tersedia).
// Password DIKETIK (tidak tampil) atau diambil dari env CREATE_USER_PASSWORD; tidak pernah lewat argumen.
//
//	go run ./tools/create-user -email andri@contoh.id -name "Andri Hermawan" -superadmin
//	go run ./tools/create-user -email budi@contoh.id -name "Budi" -project CHR -role dispatcher
//	go run ./tools/create-user -email budi@contoh.id -reset-password   # ganti password + cabut semua sesi
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"golang.org/x/term"

	authdomain "simha/core-api/internal/auth/domain"
	authrepo "simha/core-api/internal/auth/repository"
	projectrepo "simha/core-api/internal/project/repository"
	"simha/core-api/internal/shared/access"
	"simha/core-api/internal/shared/database"
	userrepo "simha/core-api/internal/user/repository"
	"simha/core-api/pkg/hash"
)

func main() {
	email := flag.String("email", "", "email (wajib)")
	name := flag.String("name", "", "nama lengkap (wajib)")
	superadmin := flag.Bool("superadmin", false, "akses lintas project")
	project := flag.String("project", "", "kode project, mis. CHR (opsional)")
	role := flag.String("role", "", "role di project: admin | dispatcher | viewer")
	reset := flag.Bool("reset-password", false, "ganti password user yang sudah ada (hanya butuh -email)")
	flag.Parse()

	*email = strings.ToLower(strings.TrimSpace(*email))
	create := *name != "" && ((*project != "" && access.ValidRole(*role)) || (*project == "" && *superadmin))
	if *email == "" || (!*reset && !create) {
		fmt.Fprintln(os.Stderr, "pemakaian: -email E -name N  (-superadmin | -project KODE -role admin|dispatcher|viewer)")
		fmt.Fprintln(os.Stderr, "           -email E -reset-password")
		os.Exit(2)
	}
	if os.Getenv("DATABASE_URL") == "" {
		fail(fmt.Errorf("DATABASE_URL kosong: muat .env dulu"))
	}

	pw := os.Getenv("CREATE_USER_PASSWORD")
	if pw == "" {
		pw = prompt("Password: ")
		if again := prompt("Ulangi password: "); again != pw {
			fail(fmt.Errorf("password tidak sama"))
		}
	}
	if err := hash.ValidatePassword(pw); err != nil {
		fail(err)
	}
	hashed, err := hash.HashPassword(pw)
	if err != nil {
		fail(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool, err := database.Open(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		fail(err)
	}
	defer pool.Close()

	if *reset {
		found, err := authrepo.New(pool).ResetPassword(ctx, *email, hashed)
		if err != nil {
			fail(fmt.Errorf("reset password: %w", err))
		}
		if !found {
			fail(fmt.Errorf("email %s tidak terdaftar", *email))
		}
		fmt.Println("password diganti & semua sesi dicabut:", *email)
		return
	}

	users := userrepo.New(pool)
	id, err := users.CreateUser(ctx, authdomain.User{Email: *email, Name: *name, PasswordHash: hashed, IsSuperadmin: *superadmin})
	if err != nil {
		fail(fmt.Errorf("buat user: %w", err))
	}
	fmt.Println("user dibuat:", id, *email, map[bool]string{true: "(superadmin)", false: ""}[*superadmin])

	if *project != "" {
		p, err := projectrepo.New(pool).GetByCode(ctx, *project)
		if err != nil || p == nil {
			fail(fmt.Errorf("project %q tidak ditemukan / nonaktif (%v)", *project, err))
		}
		if err := users.SetMembership(ctx, id, p.ID, *role, nil); err != nil {
			fail(fmt.Errorf("keanggotaan: %w", err))
		}
		fmt.Printf("anggota project %s sebagai %s\n", p.Code, *role)
	}
}

func prompt(label string) string {
	fmt.Fprint(os.Stderr, label)
	b, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr)
	if err != nil {
		fail(fmt.Errorf("baca password: %w", err))
	}
	return string(b)
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}
