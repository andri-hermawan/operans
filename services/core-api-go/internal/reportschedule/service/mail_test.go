package service

import (
	"bufio"
	"io"
	"mime"
	"mime/multipart"
	"net"
	"net/mail"
	"strconv"
	"strings"
	"testing"
)

// SMTP_FROM boleh "Nama <alamat>", "\"Nama\" <alamat>", atau alamat saja; amplop SMTP (MAIL FROM) harus alamat saja.
func TestSplitFrom(t *testing.T) {
	cases := []struct{ in, envelope string }{
		{"SIMHA <noreply@rmko.co.id>", "noreply@rmko.co.id"},
		{"\"SIMHA Report\" <noreply@rmko.co.id>", "noreply@rmko.co.id"},
		{"noreply@rmko.co.id", "noreply@rmko.co.id"},
		{"  SIMHA <noreply@rmko.co.id>  ", "noreply@rmko.co.id"},
		{`"SIMHA <noreply@rmko.co.id>"`, "noreply@rmko.co.id"}, // kutip membungkus seluruh nilai (.env)
		{`SIMHA" <noreply@rmko.co.id>`, "noreply@rmko.co.id"},  // kutip tak berpasangan (sisa pemangkasan .env)
	}
	for _, c := range cases {
		env, header := splitFrom(c.in)
		if env != c.envelope {
			t.Errorf("%q: amplop %q, seharusnya %q", c.in, env, c.envelope)
		}
		if !strings.Contains(header, c.envelope) {
			t.Errorf("%q: header %q harus memuat alamat", c.in, header)
		}
	}
}

func TestCleanAddr(t *testing.T) {
	for in, want := range map[string]string{
		"a@b.id": "a@b.id", " a@b.id ": "a@b.id", "Andi <a@b.id>": "a@b.id", `"a@b.id"`: "a@b.id",
	} {
		if got := cleanAddr(in); got != want {
			t.Errorf("%q → %q, seharusnya %q", in, got, want)
		}
	}
}

// fakeSMTP = server SMTP palsu yang mencatat percakapan, untuk memeriksa PERSIS apa yang dikirim.
type fakeSMTP struct {
	ln    net.Listener
	lines chan string // semua baris dari klien (perintah + isi email)
	data  string
}

func newFakeSMTP(t *testing.T) *fakeSMTP {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	f := &fakeSMTP{ln: ln, lines: make(chan string, 1000)}
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		r := bufio.NewReader(conn)
		say := func(s string) { _, _ = io.WriteString(conn, s+"\r\n") }
		say("220 fake ESMTP")
		var data strings.Builder
		inData := false
		for {
			line, err := r.ReadString('\n')
			if err != nil {
				return
			}
			line = strings.TrimRight(line, "\r\n")
			f.lines <- line
			if inData {
				if line == "." {
					inData = false
					f.data = data.String()
					say("250 2.0.0 OK")
				} else {
					data.WriteString(line + "\r\n")
				}
				continue
			}
			switch cmd := strings.ToUpper(line); {
			case strings.HasPrefix(cmd, "EHLO"):
				_, _ = io.WriteString(conn, "250-fake\r\n250-8BITMIME\r\n250-AUTH PLAIN\r\n250 SMTPUTF8\r\n")
			case strings.HasPrefix(cmd, "AUTH"):
				say("235 2.7.0 OK")
			case strings.HasPrefix(cmd, "MAIL FROM:"):
				// seperti Gmail: alamat harus polos di dalam <>
				if inner := cmd[len("MAIL FROM:"):]; !strings.HasPrefix(inner, "<") || strings.Count(inner, "<") != 1 {
					say("555 5.5.2 Syntax error")
					continue
				}
				say("250 2.1.0 OK")
			case strings.HasPrefix(cmd, "RCPT TO:"):
				if inner := cmd[len("RCPT TO:"):]; !strings.HasPrefix(inner, "<") || strings.Count(inner, "<") != 1 {
					say("555 5.5.2 Syntax error")
					continue
				}
				say("250 2.1.5 OK")
			case cmd == "DATA":
				inData = true
				say("354 go ahead")
			case cmd == "QUIT":
				say("221 bye")
				return
			default:
				say("500 5.5.1 unknown")
			}
		}
	}()
	t.Cleanup(func() { ln.Close() })
	return f
}

func (f *fakeSMTP) cfg(from string) SMTPConfig {
	host, port, _ := net.SplitHostPort(f.ln.Addr().String())
	p, _ := strconv.Atoi(port)
	return SMTPConfig{Host: host, Port: p, User: "andri@gmail.example", Password: "x", From: from}
}

// Kasus nyata: SMTP_FROM berbentuk "Nama <alamat>" (bukan akun yang login). Perintah SMTP harus polos, dan isi email
// harus MIME yang valid (dibaca ulang oleh net/mail + multipart).
func TestSendMailTranskrip(t *testing.T) {
	srv := newFakeSMTP(t)
	err := sendMail(srv.cfg("SIMHA <simha@gmail.example>"), []string{"Budi <budi@rmko.example>", " ani@rmko.example "},
		"[SIMHA] Alert Summary 2026-10-28 (satu hari)", "<p>Laporan — SIMHA</p>", "alert_summary_2026-10-28_harian.pdf", []byte("%PDF-1.4 uji"))
	if err != nil {
		t.Fatal(err)
	}
	var cmds []string
	for len(srv.lines) > 0 {
		cmds = append(cmds, <-srv.lines)
	}
	joined := strings.Join(cmds, "\n")
	for _, want := range []string{"MAIL FROM:<simha@gmail.example>", "RCPT TO:<budi@rmko.example>", "RCPT TO:<ani@rmko.example>"} {
		if !strings.Contains(joined, want) {
			t.Errorf("perintah %q tidak ada:\n%s", want, joined)
		}
	}

	msg, err := mail.ReadMessage(strings.NewReader(srv.data))
	if err != nil {
		t.Fatalf("isi email tidak valid: %v\n%s", err, srv.data)
	}
	if msg.Header.Get("Date") == "" || msg.Header.Get("Message-Id") == "" {
		t.Error("header Date dan Message-ID harus ada")
	}
	if from, _ := mail.ParseAddress(msg.Header.Get("From")); from == nil || from.Address != "simha@gmail.example" || from.Name != "SIMHA" {
		t.Errorf("From = %q", msg.Header.Get("From"))
	}
	mt, params, err := mime.ParseMediaType(msg.Header.Get("Content-Type"))
	if err != nil || mt != "multipart/mixed" {
		t.Fatalf("Content-Type %q %v", mt, err)
	}
	mr := multipart.NewReader(msg.Body, params["boundary"])
	var types []string
	for {
		p, err := mr.NextPart()
		if err != nil {
			break
		}
		types = append(types, strings.SplitN(p.Header.Get("Content-Type"), ";", 2)[0])
		if enc := p.Header.Get("Content-Transfer-Encoding"); enc != "base64" {
			t.Errorf("bagian %v memakai %q, seharusnya base64", types, enc)
		}
	}
	if len(types) != 2 || types[0] != "text/html" || types[1] != "application/pdf" {
		t.Errorf("bagian email: %v", types)
	}
}

// Galat harus menyebut tahap & alamat yang dipakai.
func TestSendMailGalatMenyebutTahap(t *testing.T) {
	srv := newFakeSMTP(t)
	err := sendMail(srv.cfg("SIMHA <simha@gmail.example>"), []string{"<<rusak"}, "s", "h", "f.pdf", []byte("x"))
	if err == nil || !strings.Contains(err.Error(), "RCPT TO") {
		t.Fatalf("galat harus menyebut RCPT TO: %v", err)
	}
}
