// Package teltonika menerjemahkan protokol TCP Teltonika (Codec 8 dan Codec 8 Extended).
// Hanya urusan BYTE: tidak tahu soal database, HTTP, atau aturan bisnis SIMHA.
package teltonika

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"sort"
	"time"
)

const (
	Codec8  byte = 0x08
	Codec8E byte = 0x8E

	maxDataLen = 64 * 1024 // satu paket Teltonika jauh di bawah ini; lebih besar = data rusak
)

var (
	ErrCRC       = errors.New("teltonika: CRC tidak cocok (data rusak di jalan)")
	ErrTruncated = errors.New("teltonika: paket terpotong")
)

// Record = satu AVL record (satu titik data) dari alat.
type Record struct {
	Timestamp  time.Time // UTC, dari jam alat
	Priority   uint8     // 0 low, 1 high, 2 panic
	Longitude  float64   // derajat
	Latitude   float64   // derajat
	AltitudeM  int16
	AngleDeg   uint16 // arah hadap 0..360
	Satellites uint8
	SpeedKph   uint16
	EventIOID  uint16 // ID IO yang memicu record ini (0 = record berkala)
	IO         []IOElement
}

// IOElement = satu nilai sensor/status (Teltonika menyebutnya "I/O element").
type IOElement struct {
	ID    uint16
	Size  uint8  // 1, 2, 4, 8 byte; 0 = panjang variabel (NX, khusus Codec 8E)
	Value uint64 // isi untuk Size 1/2/4/8
	Bytes []byte // isi untuk Size 0
}

// IOValue mencari nilai IO berukuran tetap berdasarkan ID.
func (r Record) IOValue(id uint16) (uint64, bool) {
	for _, e := range r.IO {
		if e.ID == id && e.Size != 0 {
			return e.Value, true
		}
	}
	return 0, false
}

// Packet = satu paket AVL lengkap (bisa berisi banyak record).
type Packet struct {
	Codec   byte
	Records []Record
	Raw     []byte // seluruh byte paket, untuk log/debug
}

// ---------------------------------------------------------------------
// MEMBACA (alat → server)
// ---------------------------------------------------------------------

// ReadIMEI membaca salam pembuka: 2 byte panjang + IMEI dalam teks ASCII.
func ReadIMEI(r io.Reader) (string, error) {
	var hdr [2]byte
	if _, err := io.ReadFull(r, hdr[:]); err != nil {
		return "", err
	}
	n := binary.BigEndian.Uint16(hdr[:])
	if n == 0 || n > 20 {
		return "", fmt.Errorf("teltonika: panjang IMEI tidak wajar (%d)", n)
	}
	buf := make([]byte, n)
	if _, err := io.ReadFull(r, buf); err != nil {
		return "", err
	}
	return string(buf), nil
}

// ReadPacket membaca satu paket AVL: preamble 0, panjang data, data, lalu CRC-16.
func ReadPacket(r io.Reader) (Packet, error) {
	var hdr [8]byte
	if _, err := io.ReadFull(r, hdr[:]); err != nil {
		return Packet{}, err
	}
	if binary.BigEndian.Uint32(hdr[0:4]) != 0 {
		return Packet{}, errors.New("teltonika: preamble bukan 0x00000000")
	}
	length := binary.BigEndian.Uint32(hdr[4:8])
	if length < 3 || length > maxDataLen {
		return Packet{}, fmt.Errorf("teltonika: panjang data tidak wajar (%d)", length)
	}

	rest := make([]byte, length+4) // data + 4 byte CRC
	if _, err := io.ReadFull(r, rest); err != nil {
		return Packet{}, ErrTruncated
	}
	data, crc := rest[:length], rest[length:]
	if uint32(CRC16(data)) != binary.BigEndian.Uint32(crc) {
		return Packet{}, ErrCRC
	}

	records, err := Decode(data)
	if err != nil {
		return Packet{}, err
	}
	return Packet{Codec: data[0], Records: records, Raw: append(hdr[:], rest...)}, nil
}

// Decode mengurai bagian data: Codec ID, jumlah record, record-record, jumlah record (lagi).
func Decode(data []byte) ([]Record, error) {
	rd := &reader{b: data}
	codec := rd.u8()
	if codec != Codec8 && codec != Codec8E {
		return nil, fmt.Errorf("teltonika: codec 0x%02X tidak didukung (hanya 0x08 dan 0x8E)", codec)
	}
	ext := codec == Codec8E

	n1 := int(rd.u8())
	records := make([]Record, 0, n1)
	for i := 0; i < n1 && rd.err == nil; i++ {
		records = append(records, decodeRecord(rd, ext))
	}
	n2 := int(rd.u8())

	switch {
	case rd.err != nil:
		return nil, rd.err
	case n1 != n2:
		return nil, fmt.Errorf("teltonika: jumlah record awal (%d) ≠ akhir (%d)", n1, n2)
	case rd.pos != len(data):
		return nil, fmt.Errorf("teltonika: ada %d byte sisa yang tidak dikenal", len(data)-rd.pos)
	}
	return records, nil
}

func decodeRecord(rd *reader, ext bool) Record {
	var rec Record
	rec.Timestamp = time.UnixMilli(int64(rd.u64())).UTC()
	rec.Priority = rd.u8()
	// GPS element (15 byte). Koordinat = bilangan bulat ×10^7, bertanda (negatif = selatan/barat).
	rec.Longitude = float64(int32(rd.u32())) / 1e7
	rec.Latitude = float64(int32(rd.u32())) / 1e7
	rec.AltitudeM = int16(rd.u16())
	rec.AngleDeg = rd.u16()
	rec.Satellites = rd.u8()
	rec.SpeedKph = rd.u16()

	// IO element. Perbedaan utama Codec 8 vs 8E ada di sini (lihat count()).
	rec.EventIOID = rd.count(ext)
	_ = rd.count(ext) // jumlah total IO (tidak diperlukan, tiap kelompok punya hitungan sendiri)
	for _, size := range []uint8{1, 2, 4, 8} {
		n := int(rd.count(ext))
		for j := 0; j < n && rd.err == nil; j++ {
			id := rd.count(ext)
			rec.IO = append(rec.IO, IOElement{ID: id, Size: size, Value: rd.uint(size)})
		}
	}
	if ext { // NX: elemen panjang variabel, hanya ada di Codec 8E
		n := int(rd.u16())
		for j := 0; j < n && rd.err == nil; j++ {
			id := rd.u16()
			l := int(rd.u16())
			rec.IO = append(rec.IO, IOElement{ID: id, Bytes: append([]byte(nil), rd.take(l)...)})
		}
	}
	return rec
}

// reader membaca byte berurutan; error pertama "menempel" sehingga kode di atas tetap ringkas.
type reader struct {
	b   []byte
	pos int
	err error
}

func (r *reader) take(n int) []byte {
	if r.err == nil && r.pos+n > len(r.b) {
		r.err = ErrTruncated
	}
	if r.err != nil {
		return make([]byte, n)
	}
	s := r.b[r.pos : r.pos+n]
	r.pos += n
	return s
}

func (r *reader) u8() uint8   { return r.take(1)[0] }
func (r *reader) u16() uint16 { return binary.BigEndian.Uint16(r.take(2)) }
func (r *reader) u32() uint32 { return binary.BigEndian.Uint32(r.take(4)) }
func (r *reader) u64() uint64 { return binary.BigEndian.Uint64(r.take(8)) }

// count membaca ID/jumlah IO: 1 byte di Codec 8, 2 byte di Codec 8 Extended.
func (r *reader) count(ext bool) uint16 {
	if ext {
		return r.u16()
	}
	return uint16(r.u8())
}

func (r *reader) uint(size uint8) uint64 {
	switch size {
	case 1:
		return uint64(r.u8())
	case 2:
		return uint64(r.u16())
	case 4:
		return uint64(r.u32())
	default:
		return r.u64()
	}
}

// CRC16 = CRC-16/IBM (polinom 0xA001) yang dipakai Teltonika.
func CRC16(data []byte) uint16 {
	var crc uint16
	for _, b := range data {
		crc ^= uint16(b)
		for i := 0; i < 8; i++ {
			if crc&1 != 0 {
				crc = crc>>1 ^ 0xA001
			} else {
				crc >>= 1
			}
		}
	}
	return crc
}

// ---------------------------------------------------------------------
// MENULIS (dipakai test dan simulator, meniru alat → server)
// ---------------------------------------------------------------------

// EncodeIMEI membuat salam pembuka seperti yang dikirim alat.
func EncodeIMEI(imei string) []byte {
	out := binary.BigEndian.AppendUint16(nil, uint16(len(imei)))
	return append(out, imei...)
}

// EncodePacket membuat paket AVL lengkap (preamble, panjang, data, CRC).
func EncodePacket(codec byte, records []Record) []byte {
	ext := codec == Codec8E
	putCount := func(b []byte, v uint16) []byte {
		if ext {
			return binary.BigEndian.AppendUint16(b, v)
		}
		return append(b, byte(v))
	}

	data := []byte{codec, byte(len(records))}
	for _, rec := range records {
		data = binary.BigEndian.AppendUint64(data, uint64(rec.Timestamp.UnixMilli()))
		data = append(data, rec.Priority)
		data = binary.BigEndian.AppendUint32(data, uint32(int32(rec.Longitude*1e7)))
		data = binary.BigEndian.AppendUint32(data, uint32(int32(rec.Latitude*1e7)))
		data = binary.BigEndian.AppendUint16(data, uint16(rec.AltitudeM))
		data = binary.BigEndian.AppendUint16(data, rec.AngleDeg)
		data = append(data, rec.Satellites)
		data = binary.BigEndian.AppendUint16(data, rec.SpeedKph)

		groups := map[uint8][]IOElement{}
		for _, e := range rec.IO {
			groups[e.Size] = append(groups[e.Size], e)
		}
		for _, g := range groups {
			sort.Slice(g, func(i, j int) bool { return g[i].ID < g[j].ID })
		}
		data = putCount(data, rec.EventIOID)
		data = putCount(data, uint16(len(rec.IO)))
		for _, size := range []uint8{1, 2, 4, 8} {
			data = putCount(data, uint16(len(groups[size])))
			for _, e := range groups[size] {
				data = putCount(data, e.ID)
				for i := int(size) - 1; i >= 0; i-- {
					data = append(data, byte(e.Value>>(8*i)))
				}
			}
		}
		if ext {
			data = binary.BigEndian.AppendUint16(data, uint16(len(groups[0])))
			for _, e := range groups[0] {
				data = binary.BigEndian.AppendUint16(data, e.ID)
				data = binary.BigEndian.AppendUint16(data, uint16(len(e.Bytes)))
				data = append(data, e.Bytes...)
			}
		}
	}
	data = append(data, byte(len(records)))

	out := make([]byte, 8, 8+len(data)+4)
	binary.BigEndian.PutUint32(out[4:8], uint32(len(data)))
	out = append(out, data...)
	return binary.BigEndian.AppendUint32(out, uint32(CRC16(data)))
}
