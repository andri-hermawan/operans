package teltonika

import (
	"bytes"
	"encoding/hex"
	"errors"
	"testing"
	"time"
)

// Contoh paket resmi dari dokumentasi Teltonika (wiki.teltonika-gps.com, "Codec").
const (
	sampleCodec8  = "000000000000003608010000016B40D8EA30010000000000000000000000000000000105021503010101425E0F01F10000601A014E0000000000000000010000C7CF"
	sampleCodec8E = "000000000000004A8E010000016B412CEE000100000000000000000000000000000000010005000100010100010011001D00010010015E2C880002000B000000003544C87A000E000000001DD7E06A00000100002994"
)

func mustHex(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(s)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestReadPacketCodec8(t *testing.T) {
	p, err := ReadPacket(bytes.NewReader(mustHex(t, sampleCodec8)))
	if err != nil {
		t.Fatal(err)
	}
	if p.Codec != Codec8 || len(p.Records) != 1 {
		t.Fatalf("codec=0x%02X records=%d", p.Codec, len(p.Records))
	}
	rec := p.Records[0]
	if got := rec.Timestamp.Format(time.RFC3339); got != "2019-06-10T10:04:46Z" {
		t.Errorf("timestamp = %s", got)
	}
	if v, ok := rec.IOValue(21); !ok || v != 3 { // GSM signal
		t.Errorf("IO 21 = %d,%v", v, ok)
	}
	if v, ok := rec.IOValue(78); !ok || v != 0 { // iButton (8 byte)
		t.Errorf("IO 78 = %d,%v", v, ok)
	}
}

func TestReadPacketCodec8E(t *testing.T) {
	p, err := ReadPacket(bytes.NewReader(mustHex(t, sampleCodec8E)))
	if err != nil {
		t.Fatal(err)
	}
	if p.Codec != Codec8E || len(p.Records) != 1 {
		t.Fatalf("codec=0x%02X records=%d", p.Codec, len(p.Records))
	}
	rec := p.Records[0]
	if rec.EventIOID != 1 {
		t.Errorf("event IO = %d", rec.EventIOID)
	}
	if v, _ := rec.IOValue(17); v != 0x1D { // ID 2 byte, nilai 2 byte
		t.Errorf("IO 17 = %d", v)
	}
	if v, _ := rec.IOValue(16); v != 0x015E2C88 { // nilai 4 byte
		t.Errorf("IO 16 = %d", v)
	}
	if v, _ := rec.IOValue(14); v != 0x1DD7E06A { // nilai 8 byte
		t.Errorf("IO 14 = %d", v)
	}
}

func TestEncodeDecodeBolakBalik(t *testing.T) {
	want := Record{
		Timestamp: time.Date(2026, 10, 22, 8, 0, 0, 0, time.UTC),
		Priority:  1, Longitude: 103.8491, Latitude: -3.47, AltitudeM: 85, AngleDeg: 270, Satellites: 12, SpeedKph: 32,
		EventIOID: 239,
		IO: []IOElement{
			{ID: 239, Size: 1, Value: 1},
			{ID: 66, Size: 2, Value: 27450},
			{ID: 16, Size: 4, Value: 123456},
			{ID: 300, Size: 8, Value: 1}, // ID > 255: hanya bisa di Codec 8E
			{ID: 385, Bytes: []byte{0xAA, 0xBB}},
		},
	}
	for _, codec := range []byte{Codec8, Codec8E} {
		rec := want
		if codec == Codec8 { // Codec 8: ID maksimal 255 dan tanpa elemen panjang variabel
			rec.IO = want.IO[:3]
		}
		p, err := ReadPacket(bytes.NewReader(EncodePacket(codec, []Record{rec, rec})))
		if err != nil {
			t.Fatalf("codec 0x%02X: %v", codec, err)
		}
		got := p.Records[1]
		if len(p.Records) != 2 || !got.Timestamp.Equal(rec.Timestamp) || got.Latitude != rec.Latitude ||
			got.Longitude != rec.Longitude || len(got.IO) != len(rec.IO) {
			t.Fatalf("codec 0x%02X: hasil berbeda: %+v", codec, got)
		}
	}
}

func TestPaketRusak(t *testing.T) {
	good := mustHex(t, sampleCodec8E)

	badCRC := append([]byte(nil), good...)
	badCRC[20] ^= 0xFF
	if _, err := ReadPacket(bytes.NewReader(badCRC)); !errors.Is(err, ErrCRC) {
		t.Errorf("CRC rusak: err = %v", err)
	}
	if _, err := ReadPacket(bytes.NewReader(good[:30])); !errors.Is(err, ErrTruncated) {
		t.Errorf("terpotong: err = %v", err)
	}
}

func TestIMEI(t *testing.T) {
	got, err := ReadIMEI(bytes.NewReader(EncodeIMEI("463719065679574")))
	if err != nil || got != "463719065679574" {
		t.Fatalf("IMEI = %q, err = %v", got, err)
	}
	if hex.EncodeToString(EncodeIMEI("356307042441013")) != "000f333536333037303432343431303133" {
		t.Fatal("format salam IMEI tidak sesuai dokumentasi")
	}
}
