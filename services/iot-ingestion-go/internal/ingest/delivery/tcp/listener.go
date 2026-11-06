// Package tcp = pintu masuk TCP untuk alat Teltonika (Codec 8 / 8 Extended).
package tcp

import (
	"bufio"
	"context"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"sync"
	"time"

	devicedomain "simha/iot-ingestion/internal/device/domain"
	"simha/iot-ingestion/internal/ingest/domain"
	"simha/iot-ingestion/internal/ingest/service"
	"simha/iot-ingestion/pkg/teltonika"
)

// DeviceFinder & Ingester = hanya kemampuan yang dibutuhkan listener.
// Dengan interface, test bisa memakai versi palsu tanpa database.
type DeviceFinder interface {
	FindDeviceByIMEI(ctx context.Context, imei string) (*devicedomain.Device, error)
}

type Ingester interface {
	Handle(ctx context.Context, dev *devicedomain.Device, e *domain.Event) (service.Result, error)
}

type Listener struct {
	Addr        string
	Devices     DeviceFinder
	Ingest      Ingester
	IdleTimeout time.Duration // putuskan alat yang diam terlalu lama (default 10 menit)
	MaxConns    int           // batas koneksi bersamaan (default 500)
	LogRaw      bool          // catat paket mentah (hex) untuk menyetel alat baru
	IO          IOSettings    // ID AVL yang bisa diganti (mis. persen fuel)
}

// Run membuka port TCP lalu melayani alat sampai ctx dibatalkan.
func (l *Listener) Run(ctx context.Context) error {
	ln, err := net.Listen("tcp", l.Addr)
	if err != nil {
		return err
	}
	slog.Info("TCP mendengarkan", "addr", ln.Addr().String())
	return l.Serve(ctx, ln)
}

// Serve menerima koneksi dari ln. Tiap alat dilayani goroutine sendiri.
func (l *Listener) Serve(ctx context.Context, ln net.Listener) error {
	stop := context.AfterFunc(ctx, func() { ln.Close() }) // ctx batal → Accept berhenti
	defer stop()

	maxConns := l.MaxConns
	if maxConns <= 0 {
		maxConns = 500
	}
	slots := make(chan struct{}, maxConns)
	var wg sync.WaitGroup
	defer wg.Wait() // tunggu semua alat selesai dilayani sebelum keluar

	for {
		conn, err := ln.Accept()
		if err != nil {
			if ctx.Err() != nil || errors.Is(err, net.ErrClosed) {
				slog.Info("TCP berhenti")
				return nil
			}
			slog.Warn("tcp: accept gagal", "err", err)
			time.Sleep(100 * time.Millisecond)
			continue
		}
		select {
		case slots <- struct{}{}:
		default:
			slog.Warn("tcp: koneksi penuh, alat ditolak", "remote", conn.RemoteAddr().String())
			conn.Close()
			continue
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() { <-slots }()
			l.serveConn(ctx, conn)
		}()
	}
}

// serveConn = percakapan dengan satu alat:
//  1. alat kirim IMEI      → server jawab 0x01 (diterima) / 0x00 (ditolak)
//  2. alat kirim paket AVL → server simpan, lalu jawab jumlah record (4 byte) = ACK
//  3. ulangi langkah 2 sampai koneksi putus
func (l *Listener) serveConn(ctx context.Context, conn net.Conn) {
	defer conn.Close()
	stop := context.AfterFunc(ctx, func() { conn.Close() }) // server berhenti → putuskan alat
	defer stop()

	remote := conn.RemoteAddr().String()
	r := bufio.NewReader(conn)

	conn.SetReadDeadline(time.Now().Add(30 * time.Second))
	imei, err := teltonika.ReadIMEI(r)
	if err != nil {
		slog.Warn("tcp: salam IMEI gagal", "remote", remote, "err", err)
		return
	}
	log := slog.With("imei", imei, "remote", remote)

	dev, err := l.Devices.FindDeviceByIMEI(ctx, imei)
	if err != nil {
		log.Error("tcp: cek IMEI gagal (database?)", "err", err) // tidak dijawab: alat akan mencoba lagi
		return
	}
	if dev == nil {
		log.Warn("tcp: IMEI tidak terdaftar atau nonaktif, ditolak")
		conn.Write([]byte{0x00})
		return
	}
	if _, err := conn.Write([]byte{0x01}); err != nil {
		return
	}
	log.Info("tcp: alat terhubung")

	idle := l.IdleTimeout
	if idle <= 0 {
		idle = 10 * time.Minute
	}
	for {
		conn.SetReadDeadline(time.Now().Add(idle))
		p, err := teltonika.ReadPacket(r)
		if err != nil {
			if errors.Is(err, io.EOF) || ctx.Err() != nil {
				log.Info("tcp: koneksi ditutup")
			} else {
				log.Warn("tcp: paket tidak bisa dibaca, koneksi ditutup", "err", err)
			}
			return
		}
		if l.LogRaw {
			log.Info("tcp: paket mentah", "hex", hex.EncodeToString(p.Raw))
		}

		if !l.store(ctx, log, dev, imei, p) {
			return // gagal simpan → TANPA ACK → alat menyimpan datanya & mengirim ulang nanti
		}

		ack := binary.BigEndian.AppendUint32(nil, uint32(len(p.Records)))
		conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
		if _, err := conn.Write(ack); err != nil {
			log.Warn("tcp: kirim ACK gagal", "err", err)
			return
		}
	}
}

// store memproses semua record dalam satu paket. false = ada kesalahan SERVER (jangan ACK).
func (l *Listener) store(ctx context.Context, log *slog.Logger, dev *devicedomain.Device, imei string, p teltonika.Packet) bool {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	var stored, duplicate, invalid int
	for _, rec := range p.Records {
		e := ToEvent(imei, p.Codec, rec, l.IO)
		res, err := l.Ingest.Handle(ctx, dev, &e)
		switch {
		case service.IsInvalid(err):
			invalid++ // sudah dicatat service; tetap di-ACK supaya alat tidak mengirim data rusak berulang-ulang
		case err != nil:
			log.Error("tcp: simpan gagal, paket tidak di-ACK", "err", err)
			return false
		case res.Duplicate:
			duplicate++
		default:
			stored++
		}
	}
	log.Info("tcp: paket diproses",
		"codec", fmt.Sprintf("0x%02X", p.Codec),
		"records", len(p.Records), "stored", stored, "duplicate", duplicate, "invalid", invalid)
	return true
}
