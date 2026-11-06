package ws

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/coder/websocket"
	"github.com/gin-gonic/gin"

	"simha/iot-ingestion/internal/shared/middleware"
)

// Kode penutupan WebSocket khusus SIMHA (rentang 4000–4999 bebas dipakai aplikasi).
const (
	closeUnauthorized websocket.StatusCode = 4401 // belum/gagal login, atau token kedaluwarsa
	closeSlow         websocket.StatusCode = 4408 // klien terlalu lambat menerima data
)

const (
	wsAuthTimeout  = 10 * time.Second // batas waktu pesan login pertama
	wsWriteTimeout = 10 * time.Second
	wsReadLimit    = 16 * 1024 // pesan dari klien hanya berisi token
)

// LiveSocketConfig = pengaturan WebSocket live.
type LiveSocketConfig struct {
	Hub            *Hub
	Verifier       *middleware.JWTVerifier
	AllowedOrigins []string      // asal halaman yang boleh membuka koneksi, mis. "localhost:8881"
	CheckEvery     time.Duration // jeda ping & cek kedaluwarsa token (bawaan 30 detik)
}

// clientMessage = pesan dari browser. Saat ini hanya {"type":"auth","token":"<access token>"}.
// Pesan auth boleh dikirim lagi kapan saja untuk memperpanjang sesi dengan token baru.
type clientMessage struct {
	Type  string `json:"type"`
	Token string `json:"token"`
}

// LiveSocket = GET /ws/live
//
//  1. browser membuka koneksi lalu mengirim {"type":"auth","token":"..."} dalam 10 detik
//  2. server menjawab {"type":"ready", ...} lalu mengirim setiap perubahan project-nya
//  3. sebelum token kedaluwarsa, browser mengirim auth lagi dengan token baru;
//     bila tidak, koneksi ditutup dengan kode 4401 dan browser menyambung ulang
func LiveSocket(cfg LiveSocketConfig) gin.HandlerFunc {
	if cfg.CheckEvery <= 0 {
		cfg.CheckEvery = 30 * time.Second
	}
	return func(c *gin.Context) {
		conn, err := websocket.Accept(c.Writer, c.Request, &websocket.AcceptOptions{OriginPatterns: cfg.AllowedOrigins})
		if err != nil {
			return // Accept sudah membalas error HTTP (mis. origin tidak diizinkan)
		}
		defer conn.CloseNow()
		conn.SetReadLimit(wsReadLimit)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// 1) Login lewat pesan pertama
		authCtx, authCancel := context.WithTimeout(ctx, wsAuthTimeout)
		p, exp, err := readAuth(authCtx, conn, cfg.Verifier)
		authCancel()
		if err != nil {
			conn.Close(closeUnauthorized, err.Error())
			return
		}
		scope := p.ProjectScope()

		sub, err := cfg.Hub.Subscribe(scope)
		if err != nil {
			conn.Close(websocket.StatusTryAgainLater, err.Error())
			return
		}
		defer cfg.Hub.Unsubscribe(sub)
		log := slog.With("user_id", p.UserID, "project_id", scope, "remote", c.ClientIP())
		log.Info("ws: terhubung", "koneksi_aktif", cfg.Hub.Count())
		defer log.Info("ws: terputus")

		ready, _ := json.Marshal(Envelope{Type: TypeReady, Data: gin.H{"project_id": scope, "token_expires_at": exp}})
		if err := write(ctx, conn, ready); err != nil {
			return
		}

		// 2) Pembaca: menerima perpanjangan token. Selesai bila klien menutup koneksi.
		var expUnix atomic.Int64
		expUnix.Store(exp.Unix())
		readerDone := make(chan error, 1)
		go func() {
			for {
				p2, exp2, err := readAuth(ctx, conn, cfg.Verifier)
				if err != nil {
					readerDone <- err
					return
				}
				if p2.SuperAdmin != p.SuperAdmin || p2.ProjectID != p.ProjectID {
					readerDone <- errors.New("token baru harus untuk project yang sama")
					return
				}
				expUnix.Store(exp2.Unix())
			}
		}()

		// 3) Penulis: kirim pesan dari hub, ping berkala, putus bila token kedaluwarsa
		ticker := time.NewTicker(cfg.CheckEvery)
		defer ticker.Stop()
		for {
			select {
			case msg := <-sub.C():
				if err := write(ctx, conn, msg); err != nil {
					return
				}
			case <-sub.Done():
				code := websocket.StatusGoingAway
				if sub.Reason() == ReasonSlow {
					code = closeSlow
				}
				conn.Close(code, sub.Reason())
				return
			case err := <-readerDone:
				var ce websocket.CloseError
				if !errors.As(err, &ce) { // bukan penutupan normal dari klien → beri tahu alasannya
					conn.Close(closeUnauthorized, err.Error())
				}
				return
			case <-ticker.C:
				if time.Now().After(time.Unix(expUnix.Load(), 0)) {
					conn.Close(closeUnauthorized, "token kedaluwarsa, kirim token baru")
					return
				}
				pingCtx, cancel := context.WithTimeout(ctx, wsWriteTimeout)
				err := conn.Ping(pingCtx)
				cancel()
				if err != nil {
					return // klien hilang tanpa menutup koneksi (mis. sinyal putus)
				}
			}
		}
	}
}

// readAuth menunggu satu pesan {"type":"auth","token":"..."} lalu memverifikasi tokennya.
func readAuth(ctx context.Context, conn *websocket.Conn, v *middleware.JWTVerifier) (middleware.Principal, time.Time, error) {
	_, data, err := conn.Read(ctx)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return middleware.Principal{}, time.Time{}, errors.New("login WebSocket tidak dikirim dalam 10 detik")
		}
		return middleware.Principal{}, time.Time{}, err
	}
	var m clientMessage
	if json.Unmarshal(data, &m) != nil || m.Type != "auth" || m.Token == "" {
		return middleware.Principal{}, time.Time{}, errors.New(`pesan harus {"type":"auth","token":"<access token>"}`)
	}
	return v.Verify(m.Token)
}

func write(ctx context.Context, conn *websocket.Conn, msg []byte) error {
	ctx, cancel := context.WithTimeout(ctx, wsWriteTimeout)
	defer cancel()
	return conn.Write(ctx, websocket.MessageText, msg)
}
