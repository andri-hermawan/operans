package tcp

import (
	"context"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	devicedomain "simha/iot-ingestion/internal/device/domain"
	"simha/iot-ingestion/internal/ingest/domain"
	"simha/iot-ingestion/internal/ingest/service"
	"simha/iot-ingestion/pkg/teltonika"
)

const imeiTerdaftar = "463719065679574"

// Database palsu: hanya mengenal satu IMEI.
type fakeDevices struct{}

func (fakeDevices) FindDeviceByIMEI(_ context.Context, imei string) (*devicedomain.Device, error) {
	if imei != imeiTerdaftar {
		return nil, nil
	}
	return &devicedomain.Device{ID: uuid.New(), IMEI: imei}, nil
}

// Service palsu: mencatat event yang diterima; bisa disuruh "database mati".
type fakeIngest struct {
	mu     sync.Mutex
	events []domain.Event
	dbDown bool
}

func (f *fakeIngest) Handle(_ context.Context, _ *devicedomain.Device, e *domain.Event) (service.Result, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.dbDown {
		return service.Result{}, errors.New("database mati")
	}
	f.events = append(f.events, *e)
	return service.Result{Inserted: true}, nil
}

func startListener(t *testing.T, ing *fakeIngest) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0") // port acak
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	l := &Listener{Devices: fakeDevices{}, Ingest: ing}
	go func() { l.Serve(ctx, ln); close(done) }()
	t.Cleanup(func() { cancel(); <-done })
	return ln.Addr().String()
}

func dial(t *testing.T, addr, imei string) (net.Conn, byte) {
	t.Helper()
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	conn.SetDeadline(time.Now().Add(5 * time.Second))
	conn.Write(teltonika.EncodeIMEI(imei))
	reply := make([]byte, 1)
	if _, err := io.ReadFull(conn, reply); err != nil {
		t.Fatal(err)
	}
	return conn, reply[0]
}

func duaRecord() []teltonika.Record {
	ts := time.Now().UTC().Truncate(time.Second)
	io := []teltonika.IOElement{{ID: 239, Size: 1, Value: 1}}
	return []teltonika.Record{
		{Timestamp: ts.Add(-30 * time.Second), Longitude: 103.849, Latitude: -3.47, Satellites: 10, SpeedKph: 20, IO: io},
		{Timestamp: ts, Longitude: 103.850, Latitude: -3.471, Satellites: 10, SpeedKph: 25, IO: io},
	}
}

func TestAlatTerdaftarDanACK(t *testing.T) {
	ing := &fakeIngest{}
	conn, reply := dial(t, startListener(t, ing), imeiTerdaftar)
	defer conn.Close()
	if reply != 0x01 {
		t.Fatalf("IMEI terdaftar seharusnya dijawab 0x01, dapat 0x%02X", reply)
	}

	conn.Write(teltonika.EncodePacket(teltonika.Codec8E, duaRecord()))
	ack := make([]byte, 4)
	if _, err := io.ReadFull(conn, ack); err != nil {
		t.Fatal(err)
	}
	if n := binary.BigEndian.Uint32(ack); n != 2 {
		t.Fatalf("ACK seharusnya 2 record, dapat %d", n)
	}
	ing.mu.Lock()
	defer ing.mu.Unlock()
	if len(ing.events) != 2 || ing.events[0].IMEI != imeiTerdaftar {
		t.Fatalf("service seharusnya menerima 2 event, dapat %d", len(ing.events))
	}
}

func TestIMEITidakTerdaftar(t *testing.T) {
	conn, reply := dial(t, startListener(t, &fakeIngest{}), "999999999999999")
	defer conn.Close()
	if reply != 0x00 {
		t.Fatalf("IMEI asing seharusnya dijawab 0x00, dapat 0x%02X", reply)
	}
}

func TestDatabaseMatiTanpaACK(t *testing.T) {
	conn, _ := dial(t, startListener(t, &fakeIngest{dbDown: true}), imeiTerdaftar)
	defer conn.Close()

	conn.Write(teltonika.EncodePacket(teltonika.Codec8E, duaRecord()))
	_, err := io.ReadFull(conn, make([]byte, 4))
	if err == nil {
		t.Fatal("saat database mati, server TIDAK boleh mengirim ACK")
	}
}

func TestPaketRusakMemutusKoneksi(t *testing.T) {
	conn, _ := dial(t, startListener(t, &fakeIngest{}), imeiTerdaftar)
	defer conn.Close()

	pkt := teltonika.EncodePacket(teltonika.Codec8E, duaRecord())
	pkt[len(pkt)-1] ^= 0xFF // rusak CRC
	conn.Write(pkt)
	if _, err := io.ReadFull(conn, make([]byte, 4)); err == nil {
		t.Fatal("paket dengan CRC salah tidak boleh di-ACK")
	}
}
