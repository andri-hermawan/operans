// keygen: membuat pasangan kunci RS256 untuk core-api.
//
//	go run ./tools/keygen
//
// keys/core_private.pem → RAHASIA, hanya dipakai core-api (JWT_PRIVATE_KEY_FILE)
// keys/core_public.pem  → boleh dibagikan; dipakai layanan lain (JWT_PUBLIC_KEY_FILE) atau lewat /.well-known/jwks.json
package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	dir := flag.String("dir", "keys", "folder tujuan")
	flag.Parse()

	privPath := filepath.Join(*dir, "core_private.pem")
	pubPath := filepath.Join(*dir, "core_public.pem")
	if _, err := os.Stat(privPath); err == nil {
		fail(fmt.Errorf("%s sudah ada; kunci baru membuat SEMUA token lama tidak berlaku — hapus manual bila memang disengaja", privPath))
	}

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		fail(err)
	}
	privDER, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		fail(err)
	}
	pubDER, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		fail(err)
	}
	if err := os.MkdirAll(*dir, 0o700); err != nil {
		fail(err)
	}
	write(privPath, "PRIVATE KEY", privDER, 0o600)
	write(pubPath, "PUBLIC KEY", pubDER, 0o644)
	fmt.Println("kunci privat :", privPath, "(RAHASIA, jangan di-commit / dibagikan)")
	fmt.Println("kunci publik :", pubPath)
}

func write(path, blockType string, der []byte, perm os.FileMode) {
	if err := os.WriteFile(path, pem.EncodeToMemory(&pem.Block{Type: blockType, Bytes: der}), perm); err != nil {
		fail(err)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}
