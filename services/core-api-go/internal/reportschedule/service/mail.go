package service

import (
	"bytes"
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"mime"
	"net"
	"net/mail"
	"net/smtp"
	"strconv"
	"strings"
	"time"
)

// SMTPConfig = server email pengirim laporan.
type SMTPConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	From     string
}

// cleanAddr merapikan satu alamat email dari data/pengaturan: buang spasi & tanda kutip yang nyasar, lalu ambil alamatnya
// saja ("SIMHA <a@b.id>" → a@b.id). Bila tidak bisa dibaca, dikembalikan apa adanya (server yang menolak, dengan pesan jelas).
func cleanAddr(s string) string {
	s = strings.Trim(strings.TrimSpace(s), `"'`)
	if a, err := mail.ParseAddress(s); err == nil {
		return a.Address
	}
	return s
}

// sendMail mengirim email HTML + satu lampiran PDF. Port 465 = TLS langsung; port lain = STARTTLS bila didukung.
// Galat selalu menyebut TAHAP yang gagal (dan alamat yang dipakai) supaya mudah dilacak.
func sendMail(cfg SMTPConfig, to []string, subject, html, fileName string, pdf []byte) error {
	from := cfg.From
	if from == "" {
		from = cfg.User
	}
	envelope, header := splitFrom(from) // MAIL FROM hanya boleh alamat; nama tampilan hanya untuk header From
	recipients := make([]string, 0, len(to))
	for _, r := range to {
		if a := cleanAddr(r); a != "" {
			recipients = append(recipients, a)
		}
	}
	addr := net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port))
	var c *smtp.Client
	var err error
	if cfg.Port == 465 {
		conn, derr := tls.Dial("tcp", addr, &tls.Config{ServerName: cfg.Host})
		if derr != nil {
			return fmt.Errorf("sambung ke %s: %w", addr, derr)
		}
		c, err = smtp.NewClient(conn, cfg.Host)
	} else {
		c, err = smtp.Dial(addr)
		if err == nil {
			if ok, _ := c.Extension("STARTTLS"); ok {
				err = c.StartTLS(&tls.Config{ServerName: cfg.Host})
			}
		}
	}
	if err != nil {
		return fmt.Errorf("sambung ke %s: %w", addr, err)
	}
	defer c.Close()
	if cfg.User != "" {
		if err := c.Auth(smtp.PlainAuth("", cfg.User, cfg.Password, cfg.Host)); err != nil {
			return fmt.Errorf("login SMTP (SMTP_USER %q): %w", cfg.User, err)
		}
	}
	if err := c.Mail(envelope); err != nil {
		return fmt.Errorf("MAIL FROM <%s> (dari SMTP_FROM): %w", envelope, err)
	}
	for _, r := range recipients {
		if err := c.Rcpt(r); err != nil {
			return fmt.Errorf("RCPT TO <%s> (penerima): %w", r, err)
		}
	}
	w, err := c.Data()
	if err != nil {
		return fmt.Errorf("DATA: %w", err)
	}
	if _, err := w.Write(buildMessage(header, recipients, subject, html, fileName, pdf)); err != nil {
		return fmt.Errorf("kirim isi email: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("isi email ditolak server (%d byte): %w", len(pdf), err)
	}
	return c.Quit()
}

// splitFrom memisahkan alamat pengirim untuk amplop SMTP ("noreply@x.id") dari tulisan header From
// ("SIMHA <noreply@x.id>"). SMTP_FROM boleh ditulis "SIMHA <noreply@x.id>" atau alamat saja. Server seperti Gmail
// menolak "MAIL FROM:<SIMHA <noreply@x.id>>" dengan galat 555 5.5.2 "Syntax error".
func splitFrom(from string) (envelope, header string) {
	from = strings.TrimSpace(from)
	if a, err := mail.ParseAddress(from); err == nil {
		return a.Address, a.String()
	}
	// nilai di .env sering bertanda kutip: "SIMHA" <noreply@x.id> atau "noreply@x.id"
	clean := strings.Trim(from, `"'`)
	if a, err := mail.ParseAddress(clean); err == nil {
		return a.Address, a.String()
	}
	if i, j := strings.LastIndex(from, "<"), strings.LastIndex(from, ">"); i >= 0 && j > i {
		addr := strings.TrimSpace(from[i+1 : j])
		return addr, addr
	}
	return clean, clean
}

// b64Lines = base64 dengan jeda baris tiap 76 karakter (batas RFC 2045).
func b64Lines(b []byte) string {
	enc := base64.StdEncoding.EncodeToString(b)
	var sb strings.Builder
	for len(enc) > 76 {
		sb.WriteString(enc[:76] + "\r\n")
		enc = enc[76:]
	}
	sb.WriteString(enc + "\r\n")
	return sb.String()
}

// buildMessage menyusun email MIME multipart (HTML + lampiran PDF), dengan header lengkap (Date, Message-ID) dan isi
// HTML ber-base64 supaya aman dari masalah karakter non-ASCII (mis. tanda "—") di server SMTP yang ketat.
func buildMessage(from string, to []string, subject, html, fileName string, pdf []byte) []byte {
	const boundary = "simha-report-boundary"
	domain := "simha.local"
	if a, err := mail.ParseAddress(from); err == nil {
		if _, d, ok := strings.Cut(a.Address, "@"); ok && d != "" {
			domain = d
		}
	}
	id := make([]byte, 8)
	_, _ = rand.Read(id)
	var b bytes.Buffer
	fmt.Fprintf(&b, "From: %s\r\nTo: %s\r\nSubject: %s\r\n", from, strings.Join(to, ", "), mime.QEncoding.Encode("utf-8", subject))
	fmt.Fprintf(&b, "Date: %s\r\nMessage-ID: <%s@%s>\r\nMIME-Version: 1.0\r\n", time.Now().Format(time.RFC1123Z), hex.EncodeToString(id), domain)
	fmt.Fprintf(&b, "Content-Type: multipart/mixed; boundary=%q\r\n\r\n", boundary)
	fmt.Fprintf(&b, "--%s\r\nContent-Type: text/html; charset=utf-8\r\nContent-Transfer-Encoding: base64\r\n\r\n%s", boundary, b64Lines([]byte(html)))
	fmt.Fprintf(&b, "--%s\r\nContent-Type: application/pdf; name=%q\r\nContent-Transfer-Encoding: base64\r\nContent-Disposition: attachment; filename=%q\r\n\r\n%s", boundary, fileName, fileName, b64Lines(pdf))
	fmt.Fprintf(&b, "--%s--\r\n", boundary)
	return b.Bytes()
}
