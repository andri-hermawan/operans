package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDotEnv(t *testing.T) {
	p := filepath.Join(t.TempDir(), ".env")
	os.WriteFile(p, []byte("# komentar\nSIMHA_UJI_A=satu\nSIMHA_UJI_B = \"dua\"\nSIMHA_UJI_C=dari-file\n"), 0o600)
	t.Setenv("SIMHA_UJI_C", "dari-sistem")
	loadDotEnv(p)
	if os.Getenv("SIMHA_UJI_A") != "satu" || os.Getenv("SIMHA_UJI_B") != "dua" {
		t.Fatalf("A=%q B=%q", os.Getenv("SIMHA_UJI_A"), os.Getenv("SIMHA_UJI_B"))
	}
	if os.Getenv("SIMHA_UJI_C") != "dari-sistem" {
		t.Fatal("variabel sistem tidak boleh ditimpa .env")
	}
	os.Unsetenv("SIMHA_UJI_A")
	os.Unsetenv("SIMHA_UJI_B")
}
